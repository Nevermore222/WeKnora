package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServeFrontendStaticPassesDesktopRemoteRequestsToAPI(t *testing.T) {
	originalWebDir, hadWebDir := os.LookupEnv("XELORA_WEB_DIR")
	t.Cleanup(func() {
		if hadWebDir {
			_ = os.Setenv("XELORA_WEB_DIR", originalWebDir)
		} else {
			_ = os.Unsetenv("XELORA_WEB_DIR")
		}
	})

	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Setenv("XELORA_WEB_DIR", webDir)

	engine := gin.New()
	serveFrontendStatic(engine)
	engine.GET("/desktop/remote/profiles", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"source": "api"})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/desktop/remote/profiles", nil)
	engine.ServeHTTP(recorder, request)

	if got := recorder.Body.String(); got != `{"source":"api"}` {
		t.Fatalf("expected desktop API response, got %q", got)
	}
}
