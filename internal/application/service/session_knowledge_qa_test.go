package service

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/event"
	"github.com/Tencent/Xelora/internal/models/asr"
	"github.com/Tencent/Xelora/internal/models/chat"
	"github.com/Tencent/Xelora/internal/models/embedding"
	"github.com/Tencent/Xelora/internal/models/rerank"
	"github.com/Tencent/Xelora/internal/models/vlm"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureChatModel struct {
	lastMessages []chat.Message
}

func (m *captureChatModel) Chat(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (*types.ChatResponse, error) {
	return nil, nil
}

func (m *captureChatModel) ChatStream(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	m.lastMessages = append([]chat.Message(nil), messages...)

	ch := make(chan types.StreamResponse, 1)
	ch <- types.StreamResponse{
		ResponseType: types.ResponseTypeAnswer,
		Content:      "ok",
		Done:         true,
	}
	close(ch)
	return ch, nil
}

func (m *captureChatModel) GetModelName() string { return "capture" }
func (m *captureChatModel) GetModelID() string   { return "capture" }

type stubModelService struct {
	chatModel chat.Chat
	models    []*types.Model
}

func (s *stubModelService) CreateModel(context.Context, *types.Model) error {
	return nil
}

func (s *stubModelService) GetModelByID(context.Context, string) (*types.Model, error) {
	return nil, nil
}

func (s *stubModelService) ListModels(context.Context) ([]*types.Model, error) {
	return s.models, nil
}

func (s *stubModelService) SetDefaultModel(context.Context, string) (*types.Model, error) {
	return nil, nil
}
func (s *stubModelService) UpdateModel(context.Context, *types.Model) error {
	return nil
}

func (s *stubModelService) DeleteModel(context.Context, string) error {
	return nil
}

func (s *stubModelService) UpdateModelCredentials(
	context.Context, string, *string, *string,
) (*types.Model, error) {
	return nil, nil
}

func (s *stubModelService) ClearModelCredential(context.Context, string, string) error {
	return nil
}

func (s *stubModelService) GetEmbeddingModel(context.Context, string) (embedding.Embedder, error) {
	return nil, nil
}

func (s *stubModelService) GetEmbeddingModelForTenant(context.Context, string, uint64) (embedding.Embedder, error) {
	return nil, nil
}

func (s *stubModelService) GetRerankModel(context.Context, string) (rerank.Reranker, error) {
	return nil, nil
}

func (s *stubModelService) GetChatModel(context.Context, string) (chat.Chat, error) {
	return s.chatModel, nil
}

func (s *stubModelService) GetVLMModel(context.Context, string) (vlm.VLM, error) {
	return nil, nil
}

func (s *stubModelService) GetASRModel(context.Context, string) (asr.ASR, error) {
	return nil, nil
}

func TestResolveAgentRerankModelIDPrefersExplicitBinding(t *testing.T) {
	svc := &sessionService{modelService: &stubModelService{models: []*types.Model{
		{ID: "default-rerank", Type: types.ModelTypeRerank, IsDefault: true},
	}}}

	got, err := svc.resolveAgentRerankModelID(context.Background(), "agent-rerank")
	require.NoError(t, err)
	assert.Equal(t, "agent-rerank", got)
}

func TestResolveAgentRerankModelIDUsesGlobalDefault(t *testing.T) {
	svc := &sessionService{modelService: &stubModelService{models: []*types.Model{
		{ID: "chat-default", Type: types.ModelTypeKnowledgeQA, IsDefault: true},
		{ID: "default-rerank", Type: types.ModelTypeRerank, IsDefault: true},
	}}}

	got, err := svc.resolveAgentRerankModelID(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "default-rerank", got)
}

func TestResolveAgentRerankModelIDReturnsEmptyWithoutDefault(t *testing.T) {
	svc := &sessionService{modelService: &stubModelService{models: []*types.Model{
		{ID: "available-rerank", Type: types.ModelTypeRerank},
	}}}

	got, err := svc.resolveAgentRerankModelID(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSelectChatModelID_PrefersTenantDefault(t *testing.T) {
	svc := &sessionService{modelService: &stubModelService{models: []*types.Model{
		{ID: "chat-first", Type: types.ModelTypeKnowledgeQA},
		{ID: "chat-default", Type: types.ModelTypeKnowledgeQA, IsDefault: true},
	}}}

	got, err := svc.selectChatModelID(modelDefaultTestContext(1, false), &types.Session{}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "chat-default", got)
}

func TestHandleModelFallback_IncludesHistoryMessages(t *testing.T) {
	chatModel := &captureChatModel{}
	svc := &sessionService{
		modelService: &stubModelService{chatModel: chatModel},
	}

	bus := event.NewEventBus()
	cm := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			SessionID:      "session-1",
			Query:          "现在还能继续讲吗？",
			ChatModelID:    "chat-model",
			FallbackPrompt: "Answer the latest user question: {{query}}",
			SummaryConfig: types.SummaryConfig{
				Temperature: 0.2,
			},
			Language: "zh-CN",
		},
		PipelineState: types.PipelineState{
			History: []*types.History{
				{
					Query:  "先介绍一下 Xelora",
					Answer: "Xelora 是一个知识库问答系统。",
				},
			},
		},
		PipelineContext: types.PipelineContext{
			EventBus: bus.AsEventBusInterface(),
		},
	}

	svc.handleModelFallback(context.Background(), cm)

	require.Len(t, chatModel.lastMessages, 3)
	assert.Equal(t, "user", chatModel.lastMessages[0].Role)
	assert.Equal(t, "先介绍一下 Xelora", chatModel.lastMessages[0].Content)
	assert.Equal(t, "assistant", chatModel.lastMessages[1].Role)
	assert.Equal(t, "Xelora 是一个知识库问答系统。", chatModel.lastMessages[1].Content)
	assert.Equal(t, "user", chatModel.lastMessages[2].Role)
	assert.Contains(t, chatModel.lastMessages[2].Content, "现在还能继续讲吗？")
}

func TestFilterSharedAgentKnowledgeBases_AllowsCrossTenantSharedWorkspaceKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-cross-tenant"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-cross-tenant",
		TenantID:   10000,
		Visibility: types.ResourceVisibilityWorkspace,
	}}
	shared := []*types.SharedKnowledgeBaseInfo{{
		KnowledgeBase:  loaded[0],
		Permission:     types.OrgRoleViewer,
		SourceTenantID: 10000,
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, shared)

	require.Equal(t, configured, got)
}

func TestFilterSharedAgentKnowledgeBases_RejectsUnsharedCrossTenantKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-unshared"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-unshared",
		TenantID:   10009,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, nil)

	require.Empty(t, got)
}

func TestFilterSharedAgentKnowledgeBases_AllowsCallerWorkspaceKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-caller-workspace"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-caller-workspace",
		TenantID:   10007,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, nil)

	require.Equal(t, configured, got)
}

func TestFilterSharedAgentKnowledgeBases_RejectsUnsharedAgentWorkspaceKB(t *testing.T) {
	agent := &types.CustomAgent{TenantID: 10000}
	configured := []string{"kb-agent-workspace"}
	loaded := []*types.KnowledgeBase{{
		ID:         "kb-agent-workspace",
		TenantID:   10000,
		Visibility: types.ResourceVisibilityWorkspace,
	}}

	got := filterSharedAgentKnowledgeBases(agent, 10007, "wangl", configured, loaded, nil)

	require.Empty(t, got)
}

type stubSearchTargetKBService struct {
	interfaces.KnowledgeBaseService
	kbs []*types.KnowledgeBase
}

func (s *stubSearchTargetKBService) GetKnowledgeBasesByIDsOnly(_ context.Context, _ []string) ([]*types.KnowledgeBase, error) {
	return s.kbs, nil
}

type recordingKBShareService struct {
	fakeKBShareService
	lastCallerTenantID uint64
}

func (s *recordingKBShareService) HasTenantKBPermission(_ context.Context, kbID string, callerTenantID uint64, _ types.TenantRole, _ types.OrgMemberRole) (bool, error) {
	s.lastCallerTenantID = callerTenantID
	return s.allowedKBs[kbID], nil
}

func TestBuildSearchTargetsForAccess_UsesCallerTenantForSharedWorkspaceKB(t *testing.T) {
	share := &recordingKBShareService{fakeKBShareService: fakeKBShareService{allowedKBs: map[string]bool{"kb-cross-tenant": true}}}
	svc := &sessionService{
		knowledgeBaseService: &stubSearchTargetKBService{kbs: []*types.KnowledgeBase{{
			ID:         "kb-cross-tenant",
			TenantID:   10009,
			Visibility: types.ResourceVisibilityWorkspace,
		}}},
		kbShareService: share,
	}
	ctx := context.WithValue(context.Background(), types.UserIDContextKey, "wangl")

	got, err := svc.buildSearchTargetsForAccess(ctx, 10000, 10007, []string{"kb-cross-tenant"}, nil)

	require.NoError(t, err)
	require.Equal(t, uint64(10007), share.lastCallerTenantID)
	require.Len(t, got, 1)
	require.Equal(t, "kb-cross-tenant", got[0].KnowledgeBaseID)
	require.Equal(t, uint64(10009), got[0].TenantID)
}

func TestBuildSearchTargetsForAccess_UsesCallerTenantForCallerWorkspaceKB(t *testing.T) {
	share := &recordingKBShareService{fakeKBShareService: fakeKBShareService{allowedKBs: map[string]bool{}}}
	svc := &sessionService{
		knowledgeBaseService: &stubSearchTargetKBService{kbs: []*types.KnowledgeBase{{
			ID:         "kb-caller-workspace",
			TenantID:   10007,
			Visibility: types.ResourceVisibilityWorkspace,
		}}},
		kbShareService: share,
	}
	ctx := context.WithValue(context.Background(), types.UserIDContextKey, "wangl")

	got, err := svc.buildSearchTargetsForAccess(ctx, 10000, 10007, []string{"kb-caller-workspace"}, nil)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, uint64(10007), got[0].TenantID)
}
