package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/gin-gonic/gin"
)

func (s *stubMemberUserService) UpdateUserCapabilities(
	ctx context.Context, userID string, canKB, canAgent, canModel bool,
) (*types.User, error) {
	if s.updateCapabilities != nil {
		return s.updateCapabilities(ctx, userID, canKB, canAgent, canModel)
	}
	u, err := s.getByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.CanCreateKnowledgeBase = canKB
	u.CanCreateAgent = canAgent
	u.CanBindModel = canModel
	return u, nil
}

func TestUpdateMemberCapabilities_RejectsSelf(t *testing.T) {
	us := &stubMemberUserService{
		getByID: func(ctx context.Context, id string) (*types.User, error) {
			return &types.User{ID: id, IsClientProvisioned: false}, nil
		},
	}
	h := newTestMemberHandler(&stubMemberService{}, us)
	body := map[string]bool{
		"can_create_knowledge_base": true,
		"can_create_agent":          true,
		"can_bind_model":            true,
	}
	w := doJSON(t, memberTestRouter(h), http.MethodPut, "/tenants/1/members/actor-1/capabilities", body, "actor-1")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("capability_denied")) {
		t.Fatalf("expected capability_denied, got %s", w.Body.String())
	}
}

func TestUpdateMemberCapabilities_RejectsClientUser(t *testing.T) {
	us := &stubMemberUserService{
		getByID: func(ctx context.Context, id string) (*types.User, error) {
			return &types.User{ID: id, IsClientProvisioned: true}, nil
		},
	}
	h := newTestMemberHandler(&stubMemberService{}, us)
	body := map[string]bool{
		"can_create_knowledge_base": true,
		"can_create_agent":          false,
		"can_bind_model":            false,
	}
	w := doJSON(t, memberTestRouter(h), http.MethodPut, "/tenants/1/members/client-1/capabilities", body, "actor-1")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("capability_denied")) {
		t.Fatalf("expected capability_denied, got %s", w.Body.String())
	}
}

func TestUpdateMemberCapabilities_Success(t *testing.T) {
	var updated bool
	us := &stubMemberUserService{
		getByID: func(ctx context.Context, id string) (*types.User, error) {
			return &types.User{
				ID:                     id,
				IsClientProvisioned:    false,
				CanCreateKnowledgeBase: false,
				CanCreateAgent:         false,
				CanBindModel:           false,
			}, nil
		},
		updateCapabilities: func(ctx context.Context, userID string, canKB, canAgent, canModel bool) (*types.User, error) {
			updated = true
			if userID != "member-2" || !canKB || !canAgent || !canModel {
				t.Fatalf("unexpected update args: %s %v %v %v", userID, canKB, canAgent, canModel)
			}
			return &types.User{
				ID:                     userID,
				CanCreateKnowledgeBase: canKB,
				CanCreateAgent:         canAgent,
				CanBindModel:           canModel,
			}, nil
		},
	}
	h := newTestMemberHandler(&stubMemberService{}, us)
	body := map[string]bool{
		"can_create_knowledge_base": true,
		"can_create_agent":          true,
		"can_bind_model":            true,
	}
	w := doJSON(t, memberTestRouter(h), http.MethodPut, "/tenants/1/members/member-2/capabilities", body, "actor-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !updated {
		t.Fatal("expected UpdateUserCapabilities to be called")
	}
	var resp gin.H
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["success"] != true {
		t.Fatalf("expected success=true, got %v", resp)
	}
}
