package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/Xelora/internal/types"
	"github.com/gin-gonic/gin"
)

type clientFeatureSettingStub struct {
	value string
}

func (s clientFeatureSettingStub) GetInt(context.Context, string, string, int64) int64 {
	return 0
}

func (s clientFeatureSettingStub) GetString(context.Context, string, string, string) string {
	return s.value
}

func (s clientFeatureSettingStub) GetBool(_ context.Context, _ string, _ string, def bool) bool {
	if s.value == "true" || s.value == "1" {
		return true
	}
	if s.value == "false" || s.value == "0" || s.value == "not-a-bool" {
		return false
	}
	return def
}

func (s clientFeatureSettingStub) GetStringList(context.Context, string, string, []string) []string {
	return nil
}

func (s clientFeatureSettingStub) List(context.Context) ([]*types.SystemSetting, error) {
	return nil, nil
}

func (s clientFeatureSettingStub) Get(context.Context, string) (*types.SystemSetting, error) {
	return nil, nil
}

func (s clientFeatureSettingStub) Update(context.Context, string, any) (*types.SystemSetting, error) {
	return nil, nil
}

func (s clientFeatureSettingStub) Reset(context.Context, string) error {
	return nil
}

func (s clientFeatureSettingStub) SubscribeRedis(context.Context) error {
	return nil
}

func callClientFeatures(h *SystemHandler) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/client-features", nil)
	h.GetClientFeatures(c)
	return w
}

func assertWorkspaceFeature(t *testing.T, w *httptest.ResponseRecorder, want bool) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", w.Code, http.StatusOK)
	}
	var got ClientFeaturesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Features["workspace_binding"] != want {
		t.Fatalf("workspace_binding=%v, want %v", got.Features["workspace_binding"], want)
	}
}

func TestGetClientFeaturesDefaultsWorkspaceBindingOff(t *testing.T) {
	h := &SystemHandler{systemSettingSvc: clientFeatureSettingStub{value: "false"}}
	assertWorkspaceFeature(t, callClientFeatures(h), false)
}

func TestGetClientFeaturesReturnsEnabledValue(t *testing.T) {
	h := &SystemHandler{systemSettingSvc: clientFeatureSettingStub{value: "true"}}
	assertWorkspaceFeature(t, callClientFeatures(h), true)
}

func TestGetClientFeaturesFailsClosedForMalformedValue(t *testing.T) {
	h := &SystemHandler{systemSettingSvc: clientFeatureSettingStub{value: "not-a-bool"}}
	assertWorkspaceFeature(t, callClientFeatures(h), false)
}
