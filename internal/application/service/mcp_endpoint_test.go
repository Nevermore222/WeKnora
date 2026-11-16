package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

type mcpEndpointRepoStub struct{ rows map[string]*types.MCPEndpoint }

func newMCPEndpointRepoStub() *mcpEndpointRepoStub {
	return &mcpEndpointRepoStub{rows: map[string]*types.MCPEndpoint{}}
}

func (r *mcpEndpointRepoStub) Create(_ context.Context, ep *types.MCPEndpoint) error {
	if ep.ID == "" {
		ep.ID = "ep-" + ep.Name
	}
	cp := *ep
	r.rows[ep.ID] = &cp
	return nil
}
func (r *mcpEndpointRepoStub) GetByID(_ context.Context, id string) (*types.MCPEndpoint, error) {
	ep, ok := r.rows[id]
	if !ok {
		return nil, nil
	}
	cp := *ep
	return &cp, nil
}
func (r *mcpEndpointRepoStub) GetByTokenHash(_ context.Context, hash string) (*types.MCPEndpoint, error) {
	for _, ep := range r.rows {
		if ep.TokenHash == hash {
			cp := *ep
			return &cp, nil
		}
	}
	return nil, nil
}
func (r *mcpEndpointRepoStub) ListByTenant(_ context.Context, tenantID uint64) ([]*types.MCPEndpoint, error) {
	var out []*types.MCPEndpoint
	for _, ep := range r.rows {
		if ep.TenantID == tenantID {
			cp := *ep
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *mcpEndpointRepoStub) Update(_ context.Context, ep *types.MCPEndpoint) error {
	cp := *ep
	r.rows[ep.ID] = &cp
	return nil
}
func (r *mcpEndpointRepoStub) Delete(_ context.Context, tenantID uint64, id string) error {
	if ep, ok := r.rows[id]; ok && ep.TenantID == tenantID {
		delete(r.rows, id)
	}
	return nil
}
func (r *mcpEndpointRepoStub) TouchLastUsed(context.Context, string) error { return nil }

func newMCPEndpointServiceForTest(repo *mcpEndpointRepoStub) *mcpEndpointService {
	return &mcpEndpointService{repo: repo}
}

func TestMCPEndpointCreateAndAuthenticate(t *testing.T) {
	repo := newMCPEndpointRepoStub()
	svc := newMCPEndpointServiceForTest(repo)
	ep, token, err := svc.Create(context.Background(), 7, &types.MCPEndpoint{
		Name: " Docs bot ", Enabled: true,
		Tools: types.StringArray{"bogus", types.MCPEndpointToolAsk, types.MCPEndpointToolSearchKnowledge},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, types.MCPEndpointTokenPrefix) {
		t.Fatalf("token %q lacks prefix", token)
	}
	if ep.TokenHash != HashMCPEndpointToken(token) || ep.TokenHash == token {
		t.Fatal("token must be hashed")
	}
	if ep.Name != "Docs bot" || ep.TenantID != 7 || ep.RateLimitPerMinute != types.DefaultMCPEndpointRateLimitPerMinute {
		t.Fatalf("unexpected endpoint: %+v", ep)
	}
	if got, err := svc.Authenticate(context.Background(), ep.ID, token); err != nil || got.ID != ep.ID {
		t.Fatalf("authenticate: %v %v", got, err)
	}
	if _, err := svc.Authenticate(context.Background(), "other", token); !errors.Is(err, ErrMCPEndpointTokenInvalid) {
		t.Fatalf("wrong endpoint token must fail: %v", err)
	}
}

func TestMCPEndpointValidationAndTenantIsolation(t *testing.T) {
	svc := newMCPEndpointServiceForTest(newMCPEndpointRepoStub())
	_, _, err := svc.Create(context.Background(), 1, &types.MCPEndpoint{Name: "", Tools: types.StringArray{"ask"}})
	if appErr, ok := apperrors.IsAppError(err); !ok || appErr.Code != apperrors.ErrBadRequest {
		t.Fatalf("empty name: %v", err)
	}
	_, _, err = svc.Create(context.Background(), 1, &types.MCPEndpoint{Name: "x", Tools: types.StringArray{"bogus"}})
	if appErr, ok := apperrors.IsAppError(err); !ok || appErr.Code != apperrors.ErrBadRequest {
		t.Fatalf("empty tool set: %v", err)
	}
	ep, _, err := svc.Create(context.Background(), 1, &types.MCPEndpoint{Name: "x", Tools: types.StringArray{"ask"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), 2, ep.ID); !errors.Is(err, ErrMCPEndpointNotFound) {
		t.Fatalf("cross-tenant get: %v", err)
	}
	disabled := false
	if _, err := svc.Update(context.Background(), 1, ep.ID, interfaces.MCPEndpointUpdate{Enabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(context.Background(), ep.ID, "bad"); !errors.Is(err, ErrMCPEndpointTokenInvalid) {
		t.Fatalf("invalid token: %v", err)
	}
}
