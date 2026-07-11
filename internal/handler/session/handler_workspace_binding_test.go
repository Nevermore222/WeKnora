package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type updateSessionWorkspaceBindingStub struct {
	interfaces.SessionService
	updated *types.Session
}

func (s *updateSessionWorkspaceBindingStub) UpdateSession(_ context.Context, session *types.Session) error {
	copy := *session
	s.updated = &copy
	return nil
}

func (s *updateSessionWorkspaceBindingStub) GetSession(_ context.Context, id string) (*types.Session, error) {
	return &types.Session{ID: id, TenantID: s.updated.TenantID, Title: s.updated.Title}, nil
}

func TestUpdateSessionWorkspaceBindingPresenceSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		body            string
		wantNil         bool
		wantWorkspaceID string
	}{
		{
			name:    "omitted preserves existing binding",
			body:    `{"title":"renamed"}`,
			wantNil: true,
		},
		{
			name:    "null clears existing binding",
			body:    `{"title":"renamed","workspace_binding":null}`,
			wantNil: false,
		},
		{
			name:    "empty object clears existing binding",
			body:    `{"title":"renamed","workspace_binding":{}}`,
			wantNil: false,
		},
		{
			name:            "object updates binding",
			body:            `{"title":"renamed","workspace_binding":{"workspace_id":"tenant:1","workspace_name":"Tenant","root_path":"/workspace"}}`,
			wantNil:         false,
			wantWorkspaceID: "tenant:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &updateSessionWorkspaceBindingStub{}
			handler := &Handler{sessionService: stub}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: "session-1"}}
			c.Set(types.TenantIDContextKey.String(), uint64(1))
			c.Request = httptest.NewRequest(http.MethodPut, "/sessions/session-1", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.UpdateSession(c)

			require.Equal(t, http.StatusOK, w.Code)
			require.NotNil(t, stub.updated)
			require.Equal(t, "session-1", stub.updated.ID)
			require.Equal(t, "renamed", stub.updated.Title)
			if tt.wantNil {
				require.Nil(t, stub.updated.WorkspaceBinding)
				return
			}
			require.NotNil(t, stub.updated.WorkspaceBinding)
			require.Equal(t, tt.wantWorkspaceID, stub.updated.WorkspaceBinding.WorkspaceID)
		})
	}
}
