package service

import (
	"context"
	"testing"

	"github.com/Tencent/Xelora/internal/application/repository"
	apperrors "github.com/Tencent/Xelora/internal/errors"
	"github.com/Tencent/Xelora/internal/types"
	"github.com/Tencent/Xelora/internal/workspace"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type sessionWorkspaceServiceStub struct {
	resolved   *workspace.Entry
	resolveErr error
}

func (s *sessionWorkspaceServiceStub) List(context.Context) ([]*workspace.Entry, error) {
	return nil, nil
}

func (s *sessionWorkspaceServiceStub) Create(context.Context, workspace.CreateInput) (*workspace.Entry, error) {
	return nil, nil
}

func (s *sessionWorkspaceServiceStub) Resolve(context.Context, string) (*workspace.Entry, error) {
	return s.resolved, s.resolveErr
}

func testSessionScopeContext(tenantID uint64, userID string) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
	if userID != "" {
		ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	}
	return ctx
}

func newTestSessionService(t *testing.T) (*sessionService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Session{}))

	return &sessionService{
		sessionRepo: repository.NewSessionRepository(db),
	}, db
}

func TestGetSessionIsScopedToCurrentUser(t *testing.T) {
	svc, db := newTestSessionService(t)
	aliceSession := &types.Session{
		TenantID: 1,
		UserID:   "alice",
		Title:    "alice private session",
	}
	require.NoError(t, db.Create(aliceSession).Error)
	bobSession := &types.Session{
		TenantID: 1,
		UserID:   "bob",
		Title:    "bob private session",
	}
	require.NoError(t, db.Create(bobSession).Error)
	legacySession := &types.Session{
		TenantID: 1,
		Title:    "legacy tenant session",
	}
	require.NoError(t, db.Create(legacySession).Error)

	_, err := svc.GetSession(testSessionScopeContext(1, "bob"), aliceSession.ID)
	require.ErrorIs(t, err, apperrors.ErrSessionNotFound)

	got, err := svc.GetSession(testSessionScopeContext(1, "bob"), bobSession.ID)
	require.NoError(t, err)
	require.Equal(t, bobSession.ID, got.ID)

	got, err = svc.GetSession(testSessionScopeContext(1, "bob"), legacySession.ID)
	require.NoError(t, err)
	require.Equal(t, legacySession.ID, got.ID)
}

func TestUpdateSessionIsScopedToCurrentUserAndAllowsNoOp(t *testing.T) {
	svc, db := newTestSessionService(t)
	aliceSession := &types.Session{
		TenantID:    1,
		UserID:      "alice",
		Title:       "alice private session",
		Description: "original description",
	}
	require.NoError(t, db.Create(aliceSession).Error)

	err := svc.UpdateSession(testSessionScopeContext(1, "bob"), &types.Session{
		ID:          aliceSession.ID,
		TenantID:    1,
		Title:       "bob update attempt",
		Description: "should not be saved",
	})
	require.ErrorIs(t, err, apperrors.ErrSessionNotFound)

	var unchanged types.Session
	require.NoError(t, db.First(&unchanged, "id = ?", aliceSession.ID).Error)
	require.Equal(t, aliceSession.Title, unchanged.Title)
	require.Equal(t, aliceSession.Description, unchanged.Description)

	err = svc.UpdateSession(testSessionScopeContext(1, "alice"), &types.Session{
		ID:          aliceSession.ID,
		TenantID:    1,
		Title:       aliceSession.Title,
		Description: aliceSession.Description,
	})
	require.NoError(t, err)
}

func TestUpdateSessionPreservesOrClearsWorkspaceBinding(t *testing.T) {
	svc, db := newTestSessionService(t)
	session := &types.Session{
		TenantID: 1,
		UserID:   "alice",
		Title:    "workspace session",
		WorkspaceBinding: &types.SessionWorkspaceBinding{
			WorkspaceID:   "tenant:1",
			WorkspaceName: "Alice workspace",
			RootPath:      "/tmp/session-workspaces/tenant-1",
			Status:        types.SessionWorkspaceBindingStatusBound,
		},
	}
	require.NoError(t, db.Create(session).Error)

	err := svc.UpdateSession(testSessionScopeContext(1, "alice"), &types.Session{
		ID:          session.ID,
		TenantID:    1,
		Title:       "rename only",
		Description: "binding should remain",
	})
	require.NoError(t, err)

	var preserved types.Session
	require.NoError(t, db.First(&preserved, "id = ?", session.ID).Error)
	require.NotNil(t, preserved.WorkspaceBinding)
	require.Equal(t, "tenant:1", preserved.WorkspaceBinding.WorkspaceID)

	err = svc.UpdateSession(testSessionScopeContext(1, "alice"), &types.Session{
		ID:               session.ID,
		TenantID:         1,
		Title:            "clear binding",
		WorkspaceBinding: &types.SessionWorkspaceBinding{},
	})
	require.NoError(t, err)

	var cleared types.Session
	require.NoError(t, db.First(&cleared, "id = ?", session.ID).Error)
	require.Nil(t, cleared.WorkspaceBinding)
}

func TestCreateSessionResolvesWorkspaceBindingServerSide(t *testing.T) {
	svc, _ := newTestSessionService(t)
	svc.workspaceService = &sessionWorkspaceServiceStub{resolved: &workspace.Entry{
		ID: "ws-1", Name: "Reports", RootPath: "/workspaces/Reports", Status: workspace.StatusAvailable,
	}}
	ctx := testSessionScopeContext(1, "alice")

	created, err := svc.CreateSession(ctx, &types.Session{
		TenantID: 1,
		UserID:   "alice",
		Title:    "workspace session",
		WorkspaceBinding: &types.SessionWorkspaceBinding{
			WorkspaceID: "ws-1",
			RootPath:    "/forged/client/path",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, created.WorkspaceBinding)
	require.Equal(t, "Reports", created.WorkspaceBinding.WorkspaceName)
	require.Equal(t, "/workspaces/Reports", created.WorkspaceBinding.RootPath)
	require.Equal(t, types.SessionWorkspaceBindingStatusBound, created.WorkspaceBinding.Status)
	require.Equal(t, "alice", created.WorkspaceBinding.BoundByUserID)
}

func TestCreateSessionRejectsUnknownWorkspace(t *testing.T) {
	svc, _ := newTestSessionService(t)
	svc.workspaceService = &sessionWorkspaceServiceStub{resolveErr: workspace.ErrNotFound}

	_, err := svc.CreateSession(testSessionScopeContext(1, "alice"), &types.Session{
		TenantID:         1,
		UserID:           "alice",
		WorkspaceBinding: &types.SessionWorkspaceBinding{WorkspaceID: "missing"},
	})

	require.ErrorContains(t, err, "workspace_not_found")
}
