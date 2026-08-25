package session

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/Xelora/internal/event"
	"github.com/Tencent/Xelora/internal/stream"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentStreamHandler_FailedCompletionPersistsStepsWithoutCompletingStream(t *testing.T) {
	manager := stream.NewMemoryStreamManager()
	message := &types.Message{ID: "msg-1", SessionID: "sess-1"}
	handler := NewAgentStreamHandler(
		context.Background(), "sess-1", "msg-1", "req-1", time.Now(),
		message, manager, event.NewEventBus(),
	)

	err := handler.handleComplete(context.Background(), event.Event{
		ID: "complete-1",
		Data: event.AgentCompleteData{
			Succeeded:  false,
			MessageID:  "msg-1",
			AgentSteps: []types.AgentStep{{Iteration: 1, Thought: "partial"}},
		},
	})
	require.NoError(t, err)
	assert.False(t, message.IsCompleted)
	assert.Len(t, message.AgentSteps, 1)

	events, _, err := manager.GetEvents(context.Background(), "sess-1", "msg-1", 0)
	require.NoError(t, err)
	assert.Empty(t, events, "failed completion must wait for the error or stop terminal event")
}

func TestAgentStreamHandler_SuccessfulCompletionCompletesMessageAndStream(t *testing.T) {
	manager := stream.NewMemoryStreamManager()
	message := &types.Message{ID: "msg-1", SessionID: "sess-1"}
	handler := NewAgentStreamHandler(
		context.Background(), "sess-1", "msg-1", "req-1", time.Now(),
		message, manager, event.NewEventBus(),
	)

	err := handler.handleComplete(context.Background(), event.Event{
		ID: "complete-1",
		Data: event.AgentCompleteData{
			Succeeded:   true,
			MessageID:   "msg-1",
			FinalAnswer: "answer",
		},
	})
	require.NoError(t, err)
	assert.True(t, message.IsCompleted)
	assert.Equal(t, "answer", message.Content)

	events, _, err := manager.GetEvents(context.Background(), "sess-1", "msg-1", 0)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Equal(t, types.ResponseTypeComplete, events[2].Type)
}
