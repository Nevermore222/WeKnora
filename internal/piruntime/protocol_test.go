package piruntime_test

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/Xelora/internal/piruntime"
	"github.com/stretchr/testify/require"
)

func TestRequestJSON_RoundTrip(t *testing.T) {
	req := piruntime.Request{
		ProtocolVersion: piruntime.ProtocolVersion,
		CorrelationID:   "c1",
		SessionID:       "s1",
		ToolName:        "workspace_read",
		Args:            json.RawMessage(`{"path":"a.txt"}`),
		Cwd:             `D:\proj`,
		TimeoutMs:       5000,
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	var got piruntime.Request
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, piruntime.ProtocolVersion, got.ProtocolVersion)
	require.Equal(t, "workspace_read", got.ToolName)
	require.Equal(t, `D:\proj`, got.Cwd)
	require.JSONEq(t, `{"path":"a.txt"}`, string(got.Args))
}

func TestResponseJSON_ErrorShape(t *testing.T) {
	resp := piruntime.Response{
		OK:           false,
		ErrorCode:    piruntime.ErrPathOutOfWorkspace,
		ErrorMessage: "escaped",
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	var got piruntime.Response
	require.NoError(t, json.Unmarshal(b, &got))
	require.False(t, got.OK)
	require.Equal(t, piruntime.ErrPathOutOfWorkspace, got.ErrorCode)
}
