package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/Tencent/Xelora/internal/middleware"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
)

type systemAdminKBStub struct {
	interfaces.KnowledgeBaseService
	kb      *types.KnowledgeBase
	byID    map[string]*types.KnowledgeBase
	visible []*types.KnowledgeBase
}

func (s *systemAdminKBStub) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	if s.byID != nil {
		if kb := s.byID[id]; kb != nil {
			return kb, nil
		}
		return nil, errors.New("not found")
	}
	if s.kb == nil {
		return nil, errors.New("not found")
	}
	return s.kb, nil
}

func (s *systemAdminKBStub) ListVisibleKnowledgeBases(_ context.Context) ([]*types.KnowledgeBase, error) {
	return s.visible, nil
}

type systemAdminKGStub struct {
	interfaces.KnowledgeService
	knowledge      *types.Knowledge
	list           []*types.Knowledge
	listedTenant   uint64
	searchedScopes []types.KnowledgeSearchScope
}

func (s *systemAdminKGStub) GetKnowledgeByIDOnly(_ context.Context, _ string) (*types.Knowledge, error) {
	if s.knowledge == nil {
		return nil, errors.New("not found")
	}
	return s.knowledge, nil
}

func (s *systemAdminKGStub) SaveKBCloneProgress(_ context.Context, _ *types.KBCloneProgress) error {
	return nil
}

func (s *systemAdminKGStub) ListKnowledgeByKnowledgeBaseID(
	ctx context.Context, _ string,
) ([]*types.Knowledge, error) {
	s.listedTenant, _ = types.TenantIDFromContext(ctx)
	return s.list, nil
}

func (s *systemAdminKGStub) SearchKnowledgeForScopes(
	_ context.Context,
	scopes []types.KnowledgeSearchScope,
	_ string, _, _ int,
	_ []string,
) ([]*types.Knowledge, bool, error) {
	s.searchedScopes = scopes
	return nil, false, nil
}

type systemAdminStoreStub struct {
	interfaces.VectorStoreService
	gotTenantID uint64
}

func (s *systemAdminStoreStub) EnvDefaultStoreView(_ context.Context) types.StoreDisplay {
	return types.DefaultStoreDisplay()
}

func (s *systemAdminStoreStub) BatchResolveStoreView(
	_ context.Context, tenantID uint64, _ []string,
) (map[string]types.StoreDisplay, error) {
	s.gotTenantID = tenantID
	return map[string]types.StoreDisplay{
		"store-10000": {
			Name:   "resource-tenant-store",
			Source: "database",
			Status: "active",
		},
	}, nil
}

func systemAdminGinContext(tenantID uint64) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
	c.Request = c.Request.WithContext(ctx)
	c.Set(types.TenantIDContextKey.String(), tenantID)
	c.Set(types.UserIDContextKey.String(), "admin-user")
	c.Set(types.SystemAdminContextKey.String(), true)
	return c, w
}

func TestValidateKnowledgeBaseAccessWithKBID_SystemAdminCrossTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	h := &KnowledgeHandler{kbService: &systemAdminKBStub{kb: &types.KnowledgeBase{
		ID:         "kb-cross-tenant",
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityPrivate,
		CreatorID:  "another-user",
	}}}
	c, _ := systemAdminGinContext(callerTenant)

	kb, _, effectiveTenantID, permission, err := h.validateKnowledgeBaseAccessWithKBID(
		c, "kb-cross-tenant",
	)
	if err != nil {
		t.Fatalf("system admin must bypass cross-tenant KB authorization: %v", err)
	}
	if kb == nil || kb.TenantID != resourceTenant {
		t.Fatalf("returned KB = %#v, want resource tenant %d", kb, resourceTenant)
	}
	if effectiveTenantID != resourceTenant {
		t.Fatalf("effective tenant = %d, want resource tenant %d", effectiveTenantID, resourceTenant)
	}
	if permission != types.OrgRoleAdmin {
		t.Fatalf("permission = %q, want %q", permission, types.OrgRoleAdmin)
	}
}

func TestValidateAndGetKnowledgeBase_SystemAdminRewritesRequestTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	h := &KnowledgeBaseHandler{service: &systemAdminKBStub{kb: &types.KnowledgeBase{
		ID:         "kb-cross-tenant",
		TenantID:   resourceTenant,
		Visibility: types.ResourceVisibilityPrivate,
		CreatorID:  "another-user",
	}}}
	c, _ := systemAdminGinContext(callerTenant)
	ctx := context.WithValue(c.Request.Context(), types.CallerTenantIDContextKey, callerTenant)
	c.Request = c.Request.WithContext(ctx)
	c.Params = gin.Params{{Key: "id", Value: "kb-cross-tenant"}}

	_, _, effectiveTenantID, permission, err := h.validateAndGetKnowledgeBase(c)
	if err != nil {
		t.Fatalf("system admin must access cross-tenant KB: %v", err)
	}
	if effectiveTenantID != resourceTenant {
		t.Fatalf("effective tenant = %d, want %d", effectiveTenantID, resourceTenant)
	}
	if permission != types.OrgRoleAdmin {
		t.Fatalf("permission = %q, want admin", permission)
	}
	if got := c.GetUint64(types.TenantIDContextKey.String()); got != resourceTenant {
		t.Fatalf("Gin tenant = %d, want resource tenant %d", got, resourceTenant)
	}
	if got, ok := types.TenantIDFromContext(c.Request.Context()); !ok || got != resourceTenant {
		t.Fatalf("request tenant = (%d, %v), want (%d, true)", got, ok, resourceTenant)
	}
	callerTenantID, ok := types.CallerTenantIDFromContext(c.Request.Context())
	if !ok || callerTenantID != callerTenant {
		t.Fatalf("caller tenant = (%d, %v), want (%d, true)", callerTenantID, ok, callerTenant)
	}
}

func TestResolveKnowledgeAndValidateKBAccess_SystemAdminUsesKnowledgeTenant(t *testing.T) {
	const (
		callerTenant    = uint64(10011)
		resourceTenant  = uint64(10000)
		knowledgeID     = "knowledge-cross-tenant"
		knowledgeBaseID = "kb-cross-tenant"
	)
	h := &KnowledgeHandler{kgService: &systemAdminKGStub{knowledge: &types.Knowledge{
		ID:              knowledgeID,
		TenantID:        resourceTenant,
		KnowledgeBaseID: knowledgeBaseID,
	}}}
	c, _ := systemAdminGinContext(callerTenant)

	knowledge, effectiveCtx, err := h.resolveKnowledgeAndValidateKBAccess(
		c, knowledgeID, types.OrgRoleAdmin,
	)
	if err != nil {
		t.Fatalf("system admin must bypass knowledge authorization: %v", err)
	}
	if knowledge == nil || knowledge.TenantID != resourceTenant {
		t.Fatalf("returned knowledge = %#v, want tenant %d", knowledge, resourceTenant)
	}
	effectiveTenantID, ok := types.TenantIDFromContext(effectiveCtx)
	if !ok || effectiveTenantID != resourceTenant {
		t.Fatalf("effective tenant = (%d, %v), want (%d, true)", effectiveTenantID, ok, resourceTenant)
	}
}

func TestBuildKBListResponse_SystemAdminResolvesStoreInResourceTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	storeID := "store-10000"
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
	storeSvc := &systemAdminStoreStub{}
	h := &KnowledgeBaseHandler{vectorStoreService: storeSvc}

	rows := h.buildKBListResponse(ctx, []*types.KnowledgeBase{{
		ID:            "kb-cross-tenant",
		TenantID:      resourceTenant,
		VectorStoreID: &storeID,
	}}, callerTenant)
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if storeSvc.gotTenantID != resourceTenant {
		t.Fatalf("store resolver tenant = %d, want resource tenant %d", storeSvc.gotTenantID, resourceTenant)
	}
	row, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("row type = %T, want map[string]interface{}", rows[0])
	}
	if got := row["vector_store_source"]; got == "shared" {
		t.Fatalf("system admin cross-tenant KB must not be projected as shared, row=%#v", row)
	}
	if got := row["vector_store_name"]; got != "resource-tenant-store" {
		t.Fatalf("vector_store_name = %v, want resource-tenant-store", got)
	}
}

type systemAdminMoveKGStub struct {
	interfaces.KnowledgeService
	knowledge *types.Knowledge
}

func (s *systemAdminMoveKGStub) GetKnowledgeByID(_ context.Context, _ string) (*types.Knowledge, error) {
	return s.knowledge, nil
}

func (s *systemAdminMoveKGStub) SaveKnowledgeMoveProgress(
	_ context.Context, _ *types.KnowledgeMoveProgress,
) error {
	return nil
}

type captureTaskEnqueuer struct {
	task *asynq.Task
}

func (s *captureTaskEnqueuer) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	s.task = task
	return &asynq.TaskInfo{ID: "asynq-move-task"}, nil
}

func TestMoveKnowledge_SystemAdminCrossTenantCarriesBothTenants(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	sourceStore := "shared-store"
	sourceKB := &types.KnowledgeBase{
		ID:               "kb-source",
		TenantID:         resourceTenant,
		Type:             "document",
		EmbeddingModelID: "embedding-1",
		VectorStoreID:    &sourceStore,
	}
	targetKB := &types.KnowledgeBase{
		ID:               "kb-target",
		TenantID:         callerTenant,
		Type:             "document",
		EmbeddingModelID: "embedding-1",
		VectorStoreID:    &sourceStore,
	}
	kbSvc := &systemAdminKBStub{byID: map[string]*types.KnowledgeBase{
		sourceKB.ID: sourceKB,
		targetKB.ID: targetKB,
	}}
	kgSvc := &systemAdminMoveKGStub{knowledge: &types.Knowledge{
		ID:              "knowledge-1",
		TenantID:        resourceTenant,
		KnowledgeBaseID: sourceKB.ID,
		ParseStatus:     types.ParseStatusCompleted,
	}}
	enqueuer := &captureTaskEnqueuer{}
	h := &KnowledgeHandler{kbService: kbSvc, kgService: kgSvc, asynqClient: enqueuer}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenant)
		c.Set(types.SystemAdminContextKey.String(), true)
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.POST("/move", h.MoveKnowledge)

	body, err := json.Marshal(MoveKnowledgeRequest{
		KnowledgeIDs: []string{"knowledge-1"},
		SourceKBID:   sourceKB.ID,
		TargetKBID:   targetKB.ID,
		Mode:         "reuse_vectors",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/move", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if enqueuer.task == nil {
		t.Fatal("move task was not enqueued")
	}
	var payload types.KnowledgeMovePayload
	if err := json.Unmarshal(enqueuer.task.Payload(), &payload); err != nil {
		t.Fatalf("decode move payload: %v", err)
	}
	if payload.SourceTenantID != resourceTenant {
		t.Fatalf("source tenant = %d, want %d", payload.SourceTenantID, resourceTenant)
	}
	if payload.TargetTenantID != callerTenant {
		t.Fatalf("target tenant = %d, want %d", payload.TargetTenantID, callerTenant)
	}
}

func TestCopyKnowledgeBase_SystemAdminCrossTenantDefaultsTargetToCaller(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	sourceKB := &types.KnowledgeBase{
		ID:               "kb-source",
		TenantID:         resourceTenant,
		Type:             "document",
		EmbeddingModelID: "embedding-1",
	}
	kbSvc := &systemAdminKBStub{byID: map[string]*types.KnowledgeBase{
		sourceKB.ID: sourceKB,
	}}
	enqueuer := &captureTaskEnqueuer{}
	h := &KnowledgeBaseHandler{
		service:          kbSvc,
		knowledgeService: &systemAdminKGStub{},
		asynqClient:      enqueuer,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenant)
		c.Set(types.SystemAdminContextKey.String(), true)
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.POST("/copy", h.CopyKnowledgeBase)

	body, err := json.Marshal(CopyKnowledgeBaseRequest{
		SourceID: sourceKB.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/copy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if enqueuer.task == nil {
		t.Fatal("clone task was not enqueued")
	}
	var payload types.KBClonePayload
	if err := json.Unmarshal(enqueuer.task.Payload(), &payload); err != nil {
		t.Fatalf("decode clone payload: %v", err)
	}
	if payload.SourceTenantID != resourceTenant {
		t.Fatalf("source tenant = %d, want %d", payload.SourceTenantID, resourceTenant)
	}
	if payload.TargetTenantID != callerTenant {
		t.Fatalf("target tenant = %d, want %d", payload.TargetTenantID, callerTenant)
	}
}

func TestClearKnowledgeBaseContents_SystemAdminUsesResourceTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	kbSvc := &systemAdminKBStub{kb: &types.KnowledgeBase{
		ID:       "kb-cross-tenant",
		TenantID: resourceTenant,
	}}
	kgSvc := &systemAdminKGStub{list: []*types.Knowledge{{
		ID:              "knowledge-1",
		TenantID:        resourceTenant,
		KnowledgeBaseID: "kb-cross-tenant",
	}}}
	enqueuer := &captureTaskEnqueuer{}
	h := &KnowledgeHandler{
		kbService:   kbSvc,
		kgService:   kgSvc,
		asynqClient: enqueuer,
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenant)
		c.Set(types.CallerTenantIDContextKey.String(), callerTenant)
		c.Set(types.SystemAdminContextKey.String(), true)
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.DELETE("/knowledge-bases/:id/knowledge", h.ClearKnowledgeBaseContents)

	req := httptest.NewRequest(http.MethodDelete, "/knowledge-bases/kb-cross-tenant/knowledge", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if kgSvc.listedTenant != resourceTenant {
		t.Fatalf("knowledge list tenant = %d, want %d", kgSvc.listedTenant, resourceTenant)
	}
	if enqueuer.task == nil {
		t.Fatal("clear task was not enqueued")
	}
	var payload types.KnowledgeListDeletePayload
	if err := json.Unmarshal(enqueuer.task.Payload(), &payload); err != nil {
		t.Fatalf("decode clear payload: %v", err)
	}
	if payload.TenantID != resourceTenant {
		t.Fatalf("clear task tenant = %d, want %d", payload.TenantID, resourceTenant)
	}
}

func TestListMoveTargets_SystemAdminSeesCrossTenantCandidates(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
		targetTenant   = uint64(10001)
	)
	source := &types.KnowledgeBase{
		ID: "kb-source", TenantID: resourceTenant, Type: "document", EmbeddingModelID: "embedding-1",
	}
	target := &types.KnowledgeBase{
		ID: "kb-target", TenantID: targetTenant, Type: "document", EmbeddingModelID: "embedding-1",
	}
	h := &KnowledgeBaseHandler{service: &systemAdminKBStub{
		byID:    map[string]*types.KnowledgeBase{source.ID: source},
		visible: []*types.KnowledgeBase{source, target},
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.CallerTenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenant)
		c.Set(types.CallerTenantIDContextKey.String(), callerTenant)
		c.Set(types.SystemAdminContextKey.String(), true)
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.GET("/knowledge-bases/:id/move-targets", h.ListMoveTargets)
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases/"+source.ID+"/move-targets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Data []*types.KnowledgeBase `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != target.ID {
		t.Fatalf("targets = %#v, want only %q", response.Data, target.ID)
	}
}

func TestDataSourceOwnedKnowledgeBase_SystemAdminCrossTenant(t *testing.T) {
	const (
		callerTenant   = uint64(10011)
		resourceTenant = uint64(10000)
	)
	h := &DataSourceHandler{kbService: &systemAdminKBStub{kb: &types.KnowledgeBase{
		ID: "kb-cross-tenant", TenantID: resourceTenant,
	}}}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)

	kb, status, message := h.getOwnedKnowledgeBase(ctx, callerTenant, "kb-cross-tenant")
	if status != http.StatusOK {
		t.Fatalf("status = %d (%s), want 200", status, message)
	}
	if kb == nil || kb.TenantID != resourceTenant {
		t.Fatalf("kb = %#v, want resource tenant %d", kb, resourceTenant)
	}
}

func TestSearchKnowledge_SystemAdminSearchesAllTenantScopes(t *testing.T) {
	const callerTenant = uint64(10011)
	kbSvc := &systemAdminKBStub{visible: []*types.KnowledgeBase{
		{ID: "kb-a", TenantID: 10000, Type: types.KnowledgeBaseTypeDocument},
		{ID: "kb-b", TenantID: 10001, Type: types.KnowledgeBaseTypeDocument},
	}}
	kgSvc := &systemAdminKGStub{}
	h := &KnowledgeHandler{kbService: kbSvc, kgService: kgSvc}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, callerTenant)
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "admin-user")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), callerTenant)
		c.Set(types.SystemAdminContextKey.String(), true)
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.GET("/knowledge/search", h.SearchKnowledge)
	req := httptest.NewRequest(http.MethodGet, "/knowledge/search?keyword=needle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if len(kgSvc.searchedScopes) != 2 {
		t.Fatalf("search scopes = %#v, want two tenant scopes", kgSvc.searchedScopes)
	}
	gotTenants := map[uint64]bool{}
	for _, scope := range kgSvc.searchedScopes {
		gotTenants[scope.TenantID] = true
	}
	if !gotTenants[10000] || !gotTenants[10001] {
		t.Fatalf("search tenants = %#v, want 10000 and 10001", gotTenants)
	}
}
