package main

import (
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
