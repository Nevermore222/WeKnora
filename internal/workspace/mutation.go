package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMutationNotFound = errors.New("mutation proposal not found")
	ErrMutationExpired  = errors.New("mutation proposal expired")
	ErrMutationToken    = errors.New("mutation token mismatch")
	ErrMutationOp       = errors.New("unsupported mutation op")
)

// MutationProposal describes a pending destructive workspace change.
type MutationProposal struct {
	ID          string    `json:"id"`
	Token       string    `json:"token"`
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`
	RootPath    string    `json:"root_path"`
	Op          string    `json:"op"` // delete | rename | move
	FromRel     string    `json:"from"`
	ToRel       string    `json:"to,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type mutationDecision struct {
	approved bool
	reason   string
}

type pendingMutation struct {
	proposal MutationProposal
	done     chan mutationDecision
}

// MutationStore holds short-lived confirmable mutations.
type MutationStore struct {
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]*pendingMutation
}

func NewMutationStore(ttl time.Duration) *MutationStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &MutationStore{ttl: ttl, items: make(map[string]*pendingMutation)}
}

var defaultMutationStore = NewMutationStore(5 * time.Minute)

// DefaultMutationStore returns the process-wide mutation store (Personal).
func DefaultMutationStore() *MutationStore { return defaultMutationStore }

func (s *MutationStore) Propose(p MutationProposal) (*MutationProposal, error) {
	p.RootPath = filepath.Clean(strings.TrimSpace(p.RootPath))
	if p.RootPath == "" {
		return nil, ErrInvalidName
	}
	fromAbs, err := resolveRelUnderRoot(p.RootPath, p.FromRel)
	if err != nil {
		return nil, err
	}
	_ = fromAbs
	switch strings.ToLower(strings.TrimSpace(p.Op)) {
	case "delete":
		p.Op = "delete"
		p.ToRel = ""
	case "rename", "move":
		p.Op = strings.ToLower(strings.TrimSpace(p.Op))
		if strings.TrimSpace(p.ToRel) == "" {
			return nil, fmt.Errorf("%w: to path required", ErrMutationOp)
		}
		if _, err := resolveRelUnderRoot(p.RootPath, p.ToRel); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrMutationOp, p.Op)
	}

	p.ID = uuid.NewString()
	token, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	p.Token = token
	p.ExpiresAt = time.Now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[p.ID] = &pendingMutation{proposal: p, done: make(chan mutationDecision, 1)}
	cp := p
	return &cp, nil
}

func (s *MutationStore) Apply(id, token string) error {
	s.mu.Lock()
	item, ok := s.items[id]
	if !ok {
		s.mu.Unlock()
		return ErrMutationNotFound
	}
	if time.Now().After(item.proposal.ExpiresAt) {
		delete(s.items, id)
		s.mu.Unlock()
		return ErrMutationExpired
	}
	if item.proposal.Token != token {
		s.mu.Unlock()
		return ErrMutationToken
	}
	p := item.proposal
	delete(s.items, id)
	done := item.done
	s.mu.Unlock()

	if err := applyMutation(p); err != nil {
		select {
		case done <- mutationDecision{approved: false, reason: err.Error()}:
		default:
		}
		return err
	}
	select {
	case done <- mutationDecision{approved: true}:
	default:
	}
	return nil
}

func (s *MutationStore) Reject(id, token string) error {
	s.mu.Lock()
	item, ok := s.items[id]
	if !ok {
		s.mu.Unlock()
		return ErrMutationNotFound
	}
	if item.proposal.Token != token {
		s.mu.Unlock()
		return ErrMutationToken
	}
	delete(s.items, id)
	done := item.done
	s.mu.Unlock()
	select {
	case done <- mutationDecision{approved: false, reason: "mutation_rejected"}:
	default:
	}
	return nil
}

// ListPending returns copies of unfinished proposals (tests / diagnostics).
func (s *MutationStore) ListPending() []MutationProposal {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MutationProposal, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item.proposal)
	}
	return out
}

// Wait blocks until the proposal is applied, rejected, expired, or ctx cancels.
func (s *MutationStore) Wait(ctx context.Context, id string) (approved bool, reason string, err error) {
	s.mu.Lock()
	item, ok := s.items[id]
	if !ok {
		s.mu.Unlock()
		return false, "", ErrMutationNotFound
	}
	done := item.done
	expires := item.proposal.ExpiresAt
	s.mu.Unlock()

	timer := time.NewTimer(time.Until(expires))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false, "canceled", ctx.Err()
	case <-timer.C:
		s.mu.Lock()
		delete(s.items, id)
		s.mu.Unlock()
		return false, "mutation_expired", ErrMutationExpired
	case d := <-done:
		return d.approved, d.reason, nil
	}
}

func applyMutation(p MutationProposal) error {
	fromAbs, err := resolveRelUnderRoot(p.RootPath, p.FromRel)
	if err != nil {
		return err
	}
	switch p.Op {
	case "delete":
		return os.RemoveAll(fromAbs)
	case "rename", "move":
		toAbs, err := resolveRelUnderRoot(p.RootPath, p.ToRel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil {
			return err
		}
		return os.Rename(fromAbs, toAbs)
	default:
		return fmt.Errorf("%w: %s", ErrMutationOp, p.Op)
	}
}

func resolveRelUnderRoot(root, rel string) (string, error) {
	cleanRel := filepath.Clean(strings.TrimSpace(rel))
	if cleanRel == "" || cleanRel == "." {
		return "", ErrPathEscape
	}
	if filepath.IsAbs(cleanRel) || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscape
	}
	abs := filepath.Join(root, cleanRel)
	if !isWithinRoot(root, abs) {
		return "", ErrPathEscape
	}
	return abs, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
