package desktopremote

import (
	"encoding/json"
	"sync"
	"time"
)

type IdentitySnapshot struct {
	Authenticated bool              `json:"authenticated"`
	UserID        string            `json:"user_id,omitempty"`
	TenantID      uint64            `json:"tenant_id,omitempty"`
	User          json.RawMessage   `json:"user,omitempty"`
	Tenant        json.RawMessage   `json:"tenant,omitempty"`
	Memberships   []json.RawMessage `json:"memberships,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at,omitempty"`
}

type Session struct {
	UserID            string
	AccessToken       string
	RefreshTokenOwner string
	AccessExpiresAt   time.Time
	Snapshot          IdentitySnapshot
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: map[string]Session{}}
}

func (s *SessionStore) Set(profileID string, session *Session) {
	if session == nil {
		s.Delete(profileID)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[profileID] = cloneSession(*session)
}

func (s *SessionStore) Get(profileID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[profileID]
	if !ok {
		return nil, false
	}
	cloned := cloneSession(session)
	return &cloned, true
}

func (s *SessionStore) Delete(profileID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, profileID)
}

func (s *SessionStore) Snapshot(profileID string) IdentitySnapshot {
	session, ok := s.Get(profileID)
	if !ok {
		return IdentitySnapshot{}
	}
	return cloneIdentitySnapshot(session.Snapshot)
}

func cloneSession(session Session) Session {
	session.Snapshot = cloneIdentitySnapshot(session.Snapshot)
	return session
}

func cloneIdentitySnapshot(snapshot IdentitySnapshot) IdentitySnapshot {
	snapshot.User = cloneRawMessage(snapshot.User)
	snapshot.Tenant = cloneRawMessage(snapshot.Tenant)
	if snapshot.Memberships != nil {
		memberships := make([]json.RawMessage, len(snapshot.Memberships))
		for i := range snapshot.Memberships {
			memberships[i] = cloneRawMessage(snapshot.Memberships[i])
		}
		snapshot.Memberships = memberships
	}
	return snapshot
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}
