package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetDesktopCapabilitiesIsPublicAndStable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/system/capabilities", GetSystemCapabilities)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		"/api/v1/system/capabilities", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var got SystemCapabilitiesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.APIContractMajor != 1 || got.APIContractMinor < 0 {
		t.Fatalf("unexpected contract: %+v", got)
	}

	required := map[string]bool{
		"tenant_rbac":      false,
		"organizations":    false,
		"shared_resources": false,
		"sse_chat":         false,
	}
	for _, feature := range got.Features {
		if _, ok := required[feature]; ok {
			required[feature] = true
		}
	}
	for feature, present := range required {
		if !present {
			t.Fatalf("missing feature %q", feature)
		}
	}
}
