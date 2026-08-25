package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDesktopEnvPrefersExecutablePersonalEnvOverCurrentWorkingDirectory(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	originalServerURL, hadServerURL := os.LookupEnv("XELORA_PERSONAL_SERVER_URL")
	_ = os.Unsetenv("XELORA_PERSONAL_SERVER_URL")
	t.Cleanup(func() {
		if hadServerURL {
			_ = os.Setenv("XELORA_PERSONAL_SERVER_URL", originalServerURL)
		} else {
			_ = os.Unsetenv("XELORA_PERSONAL_SERVER_URL")
		}
	})

	root := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	exeDir := filepath.Join(root, "dist", "personal")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("XELORA_PERSONAL_SERVER_URL=http://wrong.example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exeDir, ".env.personal"), []byte("XELORA_PERSONAL_SERVER_URL=http://localhost:8080\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	loadDesktopEnv(filepath.Join(exeDir, "Xelora Personal.exe"))

	if got := os.Getenv("XELORA_PERSONAL_SERVER_URL"); got != "http://localhost:8080" {
		t.Fatalf("expected executable .env.personal to win, got %q", got)
	}
}

func TestConfigureDesktopWebDirUsesExecutableDirectory(t *testing.T) {
	originalWebDir, hadWebDir := os.LookupEnv("XELORA_WEB_DIR")
	_ = os.Unsetenv("XELORA_WEB_DIR")
	t.Cleanup(func() {
		if hadWebDir {
			_ = os.Setenv("XELORA_WEB_DIR", originalWebDir)
		} else {
			_ = os.Unsetenv("XELORA_WEB_DIR")
		}
	})

	exeDir := filepath.Join(t.TempDir(), "dist", "personal")
	webDir := filepath.Join(exeDir, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	configureDesktopWebDir(filepath.Join(exeDir, "Xelora Personal.exe"))

	if got := os.Getenv("XELORA_WEB_DIR"); got != webDir {
		t.Fatalf("expected executable web directory %q, got %q", webDir, got)
	}
}

func TestDefaultEnterpriseServerURLFallsBackToLocalhost(t *testing.T) {
	originalServerURL, hadServerURL := os.LookupEnv("XELORA_PERSONAL_SERVER_URL")
	_ = os.Unsetenv("XELORA_PERSONAL_SERVER_URL")
	t.Cleanup(func() {
		if hadServerURL {
			_ = os.Setenv("XELORA_PERSONAL_SERVER_URL", originalServerURL)
		} else {
			_ = os.Unsetenv("XELORA_PERSONAL_SERVER_URL")
		}
	})

	if got := defaultEnterpriseServerURL(); got != "http://localhost:8080" {
		t.Fatalf("expected localhost fallback URL, got %q", got)
	}
}

func TestDefaultEnterpriseServerURLPrefersPackagedOrigin(t *testing.T) {
	originalPackaged := packagedEnterpriseServerURL
	packagedEnterpriseServerURL = "http://xelora-server.internal:8080"
	t.Cleanup(func() { packagedEnterpriseServerURL = originalPackaged })
	t.Setenv("XELORA_PERSONAL_SERVER_URL", "http://localhost:8080")

	if got := defaultEnterpriseServerURL(); got != "http://xelora-server.internal:8080" {
		t.Fatalf("expected packaged server URL to win, got %q", got)
	}
}

func TestLocalEnterpriseServerFallbackUsesLoopbackOnlyForLocalIP(t *testing.T) {
	localIPs := []net.IP{net.ParseIP("192.168.2.61")}

	if got := localEnterpriseServerFallbackURL("http://192.168.2.61:8080", localIPs); got != "http://127.0.0.1:8080" {
		t.Fatalf("expected local server fallback to loopback, got %q", got)
	}
	if got := localEnterpriseServerFallbackURL("http://192.168.2.62:8080", localIPs); got != "" {
		t.Fatalf("expected no fallback for a remote IP, got %q", got)
	}
	if got := localEnterpriseServerFallbackURL("http://xelora-server.internal:8080", localIPs); got != "" {
		t.Fatalf("expected no fallback for a hostname, got %q", got)
	}
}
