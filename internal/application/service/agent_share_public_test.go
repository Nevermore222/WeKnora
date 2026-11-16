package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/Xelora/internal/application/repository"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

// pubAgentNoShareRepo always reports "no organization share row", which is the
// state of a public agent that was never shared into any organization.
type pubAgentNoShareRepo struct {
	interfaces.AgentShareRepository
}

func (pubAgentNoShareRepo) GetShareByAgentIDForTenant(
	_ context.Context, _ uint64, _ string, _ uint64,
) (*types.AgentShare, error) {
	return nil, repository.ErrAgentShareNotFound
}

// pubAgentLookupRepo returns one fixed agent regardless of tenant scoping.
type pubAgentLookupRepo struct {
	interfaces.CustomAgentRepository
	agent *types.CustomAgent
}

func (r pubAgentLookupRepo) GetAgentByIDOnly(_ context.Context, _ string) (*types.CustomAgent, error) {
	if r.agent == nil {
		return nil, repository.ErrCustomAgentNotFound
	}
	return r.agent, nil
}

// pubAgentDisabledRepo returns a fixed tenant-disabled list.
type pubAgentDisabledRepo struct {
	interfaces.TenantDisabledSharedAgentRepository
	rows []*types.TenantDisabledSharedAgent
}

func (r pubAgentDisabledRepo) ListByTenantID(
	_ context.Context, _ uint64,
) ([]*types.TenantDisabledSharedAgent, error) {
	return r.rows, nil
}

func pubAgentFixtures(
	agent *types.CustomAgent,
	disabled []*types.TenantDisabledSharedAgent,
) *agentShareService {
	return &agentShareService{
		shareRepo:    pubAgentNoShareRepo{},
		disabledRepo: pubAgentDisabledRepo{rows: disabled},
		agentRepo:    pubAgentLookupRepo{agent: agent},
	}
}

// A system-public agent is reachable by every workspace read-only, the same way
// customAgentService.GetAgentByID and session/qa.go's resolveAgent treat it.
// Agent-scoped endpoints (?agent_id=) resolve through GetSharedAgentForTenant,
// so without this fallback a public agent can be chatted with while its bound
// knowledge base list returns 403.
func TestGetSharedAgentForTenant_PublicAgentWithoutShareRecord(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	s := pubAgentFixtures(&types.CustomAgent{
		ID:         "agent-public",
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityPublic,
	}, nil)

	agent, err := s.GetSharedAgentForTenant(
		context.Background(), callerTenant, types.TenantRoleOwner, "agent-public",
	)
	if err != nil {
		t.Fatalf("public agent must resolve without an organization share row: %v", err)
	}
	if agent == nil || agent.TenantID != resourceTenant {
		t.Fatalf("agent = %#v, want source tenant %d", agent, resourceTenant)
	}
	if !agent.ReadOnly {
		t.Fatal("public agent must be returned read-only for a foreign workspace")
	}
}

// A tenant that turned the agent off in its chat dropdown must keep it off: the
// public fallback honours tenant_disabled_shared_agents like the org-share path.
func TestGetSharedAgentForTenant_PublicAgentDisabledByTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	s := pubAgentFixtures(&types.CustomAgent{
		ID:         "agent-public",
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityPublic,
	}, []*types.TenantDisabledSharedAgent{{
		TenantID:       callerTenant,
		AgentID:        "agent-public",
		SourceTenantID: resourceTenant,
	}})

	if _, err := s.GetSharedAgentForTenant(
		context.Background(), callerTenant, types.TenantRoleOwner, "agent-public",
	); !errors.Is(err, ErrAgentSharePermission) {
		t.Fatalf("err = %v, want %v", err, ErrAgentSharePermission)
	}
}

// Regression guard: a workspace-visible agent from another tenant is still not
// reachable without an organization share row. The public fallback must not
// widen into a general cross-workspace agent bypass.
func TestGetSharedAgentForTenant_WorkspaceAgentWithoutShareRecordDenied(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	s := pubAgentFixtures(&types.CustomAgent{
		ID:         "agent-workspace",
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityWorkspace,
	}, nil)

	if _, err := s.GetSharedAgentForTenant(
		context.Background(), callerTenant, types.TenantRoleOwner, "agent-workspace",
	); !errors.Is(err, ErrAgentSharePermission) {
		t.Fatalf("err = %v, want %v", err, ErrAgentSharePermission)
	}
}
