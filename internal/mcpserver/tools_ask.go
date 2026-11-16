package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/Xelora/internal/event"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

const askChannel = "mcp"

func askTool() mcp.Tool {
	return mcp.NewTool(types.MCPEndpointToolAsk,
		mcp.WithDescription("Ask the configured workspace agent and return a synthesized answer with citations."),
		mcp.WithString("question", mcp.Required()),
		mcp.WithString("session_id"),
		mcp.WithArray("knowledge_base_ids", mcp.WithStringItems()),
		mcp.WithReadOnlyHintAnnotation(false),
	)
}

func (s *Server) handleAsk(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ep, err := endpointFromContext(ctx)
	if err != nil {
		return mcp.NewToolResultError("unauthorized"), nil
	}
	question := strings.TrimSpace(req.GetString("question", ""))
	if question == "" {
		return mcp.NewToolResultError("question is required"), nil
	}
	if s.sessionService == nil || s.messageService == nil || s.agentService == nil {
		return mcp.NewToolResultError("ask service is unavailable"), nil
	}
	agentID := strings.TrimSpace(ep.DefaultAgentID)
	if agentID == "" {
		agentID = types.BuiltinQuickAnswerID
	}
	agent, err := s.agentService.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil || !types.MCPEndpointAgentAllowed(agent, ep.TenantID) {
		return mcp.NewToolResultError("configured agent is unavailable"), nil
	}
	kbIDs, err := s.resolveAskKnowledgeBases(ctx, ep, req.GetStringSlice("knowledge_base_ids", nil))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	userID, _ := types.UserIDFromContext(ctx)
	if userID == "" {
		userID = "mcp-" + ep.ID
	}
	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	var session *types.Session
	if sessionID != "" {
		session, err = s.sessionService.GetSession(ctx, sessionID)
		if err != nil || session == nil || session.UserID != userID {
			return mcp.NewToolResultError("session was not found on this endpoint"), nil
		}
	} else {
		session, err = s.sessionService.CreateSession(ctx, &types.Session{TenantID: ep.TenantID, UserID: userID, Title: question, Description: "MCP endpoint " + ep.Name})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create session", err), nil
		}
	}
	requestID := uuid.NewString()
	userMessage, err := s.messageService.CreateMessage(ctx, &types.Message{SessionID: session.ID, Role: "user", Content: question, RequestID: requestID, IsCompleted: true, Channel: askChannel, CreatedAt: time.Now()})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create user message", err), nil
	}
	assistant, err := s.messageService.CreateMessage(ctx, &types.Message{SessionID: session.ID, Role: "assistant", RequestID: requestID, IsCompleted: false, Channel: askChannel, CreatedAt: time.Now()})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to create assistant message", err), nil
	}

	var mu sync.Mutex
	answer := strings.Builder{}
	var refs []interface{}
	bus := event.NewEventBus()
	bus.On(event.EventAgentFinalAnswer, func(_ context.Context, evt event.Event) error {
		data, ok := evt.Data.(event.AgentFinalAnswerData)
		if !ok {
			return nil
		}
		mu.Lock()
		answer.WriteString(data.Content)
		mu.Unlock()
		return nil
	})
	bus.On(event.EventAgentComplete, func(_ context.Context, evt event.Event) error {
		data, ok := evt.Data.(event.AgentCompleteData)
		if !ok {
			return nil
		}
		mu.Lock()
		if data.FinalAnswer != "" {
			answer.Reset()
			answer.WriteString(data.FinalAnswer)
		}
		refs = data.KnowledgeRefs
		mu.Unlock()
		return nil
	})
	qaErr := s.sessionService.AgentQA(ctx, &types.QARequest{Session: session, Query: question, AssistantMessageID: assistant.ID, UserMessageID: userMessage.ID, CustomAgent: agent, KnowledgeBaseIDs: kbIDs}, bus)
	if qaErr != nil {
		return mcp.NewToolResultErrorFromErr("failed to answer", qaErr), nil
	}
	mu.Lock()
	text := strings.TrimSpace(answer.String())
	outRefs := refs
	mu.Unlock()
	if text == "" {
		text = "The agent returned an empty answer."
	}
	return mcp.NewToolResultStructured(map[string]any{"answer": text, "session_id": session.ID, "agent_id": agent.ID, "references": outRefs}, fmt.Sprintf("%s\n\nsession_id: %s", text, session.ID)), nil
}

func (s *Server) resolveAskKnowledgeBases(ctx context.Context, ep *types.MCPEndpoint, requested []string) ([]string, error) {
	if len(requested) > 0 {
		kbs, err := s.selectKnowledgeBases(ctx, ep, requested)
		if err != nil {
			return nil, err
		}
		return knowledgeBaseIDs(kbs), nil
	}
	kbs, err := s.allowedKnowledgeBases(ctx, ep)
	if err != nil {
		return nil, err
	}
	return knowledgeBaseIDs(kbs), nil
}
