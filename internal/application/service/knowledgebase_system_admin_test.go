package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

func ctxWithSystemAdminTenant(tenantID uint64) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
	return context.WithValue(ctx, types.SystemAdminContextKey, true)
}

func TestCopyKnowledgeBase_SystemAdminCrossTenantExistingTarget(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	repo := newFakeKBRepo()
	repo.rows["src"] = &types.KnowledgeBase{
		ID: "src", Name: "src", TenantID: resourceTenant, EmbeddingModelID: "embed-1",
	}
	repo.rows["dst"] = &types.KnowledgeBase{
		ID: "dst", Name: "dst", TenantID: callerTenant, EmbeddingModelID: "embed-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	src, dst, err := svc.CopyKnowledgeBase(ctxWithSystemAdminTenant(callerTenant), "src", "dst")
	require.NoError(t, err)
	require.Equal(t, resourceTenant, src.TenantID)
	require.Equal(t, callerTenant, dst.TenantID)
}

func TestCopyKnowledgeBase_SystemAdminNewTargetUsesCallerTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	repo := newFakeKBRepo()
	repo.rows["src"] = &types.KnowledgeBase{
		ID: "src", Name: "src", TenantID: resourceTenant, EmbeddingModelID: "embed-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	src, dst, err := svc.CopyKnowledgeBase(ctxWithSystemAdminTenant(callerTenant), "src", "")
	require.NoError(t, err)
	require.Equal(t, resourceTenant, src.TenantID)
	require.Equal(t, callerTenant, dst.TenantID)
	require.Equal(t, callerTenant, repo.rows[dst.ID].TenantID)
}

func TestCopyKnowledgeBase_WorkerContextUsesSourceAndCallerTenantsSeparately(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	repo := newFakeKBRepo()
	repo.rows["src"] = &types.KnowledgeBase{
		ID: "src", Name: "src", TenantID: resourceTenant, EmbeddingModelID: "embed-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, resourceTenant)
	ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, callerTenant)

	src, dst, err := svc.CopyKnowledgeBase(ctx, "src", "")
	require.NoError(t, err)
	require.Equal(t, resourceTenant, src.TenantID)
	require.Equal(t, callerTenant, dst.TenantID)
}

type systemAdminTenantRepoStub struct {
	interfaces.TenantRepository
	tenants map[uint64]*types.Tenant
}

func (s *systemAdminTenantRepoStub) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	return s.tenants[id], nil
}

type systemAdminShareRepoStub struct {
	interfaces.KBShareRepository
}

func (s *systemAdminShareRepoStub) DeleteByKnowledgeBaseID(_ context.Context, _ string) error {
	return nil
}

type systemAdminTaskStub struct {
	task *asynq.Task
}

func (s *systemAdminTaskStub) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	s.task = task
	return &asynq.TaskInfo{ID: "task-1"}, nil
}

func TestDeleteKnowledgeBase_SystemAdminUsesResourceTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	repo := newFakeKBRepo()
	repo.rows["kb-cross-tenant"] = &types.KnowledgeBase{
		ID: "kb-cross-tenant", TenantID: resourceTenant,
	}
	taskSvc := &systemAdminTaskStub{}
	svc := &knowledgeBaseService{
		repo:        repo,
		shareRepo:   &systemAdminShareRepoStub{},
		tenantRepo:  &systemAdminTenantRepoStub{tenants: map[uint64]*types.Tenant{resourceTenant: {ID: resourceTenant}}},
		asynqClient: taskSvc,
	}

	ctx := ctxWithSystemAdminTenant(callerTenant)
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, &types.Tenant{ID: callerTenant})
	err := svc.DeleteKnowledgeBase(ctx, "kb-cross-tenant")
	require.NoError(t, err)
	require.NotNil(t, taskSvc.task)

	var payload types.KBDeletePayload
	require.NoError(t, json.Unmarshal(taskSvc.task.Payload(), &payload))
	require.Equal(t, resourceTenant, payload.SourceTenantID)
	require.Equal(t, resourceTenant, payload.SourceTenant())
}

type systemAdminKnowledgeRepoStub struct {
	interfaces.KnowledgeRepository
	updated *types.Knowledge
}

func (s *systemAdminKnowledgeRepoStub) UpdateKnowledge(_ context.Context, knowledge *types.Knowledge) error {
	s.updated = knowledge
	return nil
}

type systemAdminChunkRepoStub struct {
	interfaces.ChunkRepository
	moveSourceTenant uint64
	moveTargetTenant uint64
	moveTargetKBID   string
}

func (s *systemAdminChunkRepoStub) ListChunksByKnowledgeID(
	_ context.Context, _ uint64, _ string,
) ([]*types.Chunk, error) {
	return nil, nil
}

func (s *systemAdminChunkRepoStub) MoveChunksToKnowledgeBase(
	_ context.Context,
	sourceTenantID, targetTenantID uint64,
	_ string, targetKBID string,
) error {
	s.moveSourceTenant = sourceTenantID
	s.moveTargetTenant = targetTenantID
	s.moveTargetKBID = targetKBID
	return nil
}

func TestMoveKnowledgeReparse_SystemAdminSwitchesKnowledgeToTargetTenant(t *testing.T) {
	const (
		sourceTenant = uint64(10000)
		targetTenant = uint64(10011)
	)
	repo := &systemAdminKnowledgeRepoStub{}
	taskSvc := &systemAdminTaskStub{}
	svc := &knowledgeService{repo: repo, task: taskSvc}
	sourceCtx := context.WithValue(context.Background(), types.TenantIDContextKey, sourceTenant)
	sourceCtx = context.WithValue(sourceCtx, types.TenantInfoContextKey, &types.Tenant{ID: sourceTenant})
	targetCtx := context.WithValue(context.Background(), types.TenantIDContextKey, targetTenant)
	targetCtx = context.WithValue(targetCtx, types.TenantInfoContextKey, &types.Tenant{ID: targetTenant})
	knowledge := &types.Knowledge{
		ID:               "knowledge-1",
		TenantID:         sourceTenant,
		KnowledgeBaseID:  "kb-source",
		Type:             "file",
		FileType:         "txt",
		FileName:         "fixture.txt",
		FilePath:         "tenant-10000/fixture.txt",
		ParseStatus:      types.ManualKnowledgeStatusDraft,
		StorageSize:      0,
		EmbeddingModelID: "embedding-1",
	}
	targetKB := &types.KnowledgeBase{
		ID:               "kb-target",
		TenantID:         targetTenant,
		Type:             "document",
		EmbeddingModelID: "embedding-1",
	}

	err := svc.moveKnowledgeReparse(sourceCtx, targetCtx, knowledge, nil, targetKB)
	require.NoError(t, err)
	require.Equal(t, targetTenant, repo.updated.TenantID)
	require.Equal(t, targetKB.ID, repo.updated.KnowledgeBaseID)
	require.NotNil(t, taskSvc.task)

	var payload types.DocumentProcessPayload
	require.NoError(t, json.Unmarshal(taskSvc.task.Payload(), &payload))
	require.Equal(t, targetTenant, payload.TenantID)
	require.Equal(t, targetKB.ID, payload.KnowledgeBaseID)
}

func TestWithKnowledgeBaseTenantContext_PreservesCallerTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	svc := &knowledgeService{tenantRepo: &systemAdminTenantRepoStub{tenants: map[uint64]*types.Tenant{
		resourceTenant: {ID: resourceTenant},
	}}}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, &types.Tenant{ID: callerTenant})

	resourceCtx, err := svc.withKnowledgeBaseTenantContext(ctx, &types.KnowledgeBase{TenantID: resourceTenant})
	require.NoError(t, err)
	require.Equal(t, resourceTenant, types.MustTenantIDFromContext(resourceCtx))
	gotCallerTenant, ok := types.CallerTenantIDFromContext(resourceCtx)
	require.True(t, ok)
	require.Equal(t, callerTenant, gotCallerTenant)
	tenantInfo, ok := types.TenantInfoFromContext(resourceCtx)
	require.True(t, ok)
	require.Equal(t, resourceTenant, tenantInfo.ID)
}

func TestMoveKnowledgeReuseVectors_SystemAdminMovesChunkTenants(t *testing.T) {
	const (
		sourceTenant = uint64(10000)
		targetTenant = uint64(10011)
	)
	storeID := "shared-store"
	repo := &systemAdminKnowledgeRepoStub{}
	chunks := &systemAdminChunkRepoStub{}
	svc := &knowledgeService{repo: repo, chunkRepo: chunks}
	sourceCtx := context.WithValue(context.Background(), types.TenantIDContextKey, sourceTenant)
	targetCtx := context.WithValue(context.Background(), types.TenantIDContextKey, targetTenant)
	knowledge := &types.Knowledge{
		ID: "knowledge-1", TenantID: sourceTenant, KnowledgeBaseID: "kb-source",
		ParseStatus: types.ParseStatusProcessing,
	}
	sourceKB := &types.KnowledgeBase{
		ID: "kb-source", TenantID: sourceTenant, Type: "document", VectorStoreID: &storeID,
	}
	targetKB := &types.KnowledgeBase{
		ID: "kb-target", TenantID: targetTenant, Type: "document", VectorStoreID: &storeID,
	}

	err := svc.moveKnowledgeReuseVectors(sourceCtx, targetCtx, knowledge, sourceKB, targetKB)
	require.NoError(t, err)
	require.Equal(t, sourceTenant, chunks.moveSourceTenant)
	require.Equal(t, targetTenant, chunks.moveTargetTenant)
	require.Equal(t, targetKB.ID, chunks.moveTargetKBID)
	require.Equal(t, targetTenant, repo.updated.TenantID)
	require.Equal(t, targetKB.ID, repo.updated.KnowledgeBaseID)
}
