package main

import (
	"testing"

	"github.com/Tencent/Xelora/internal/branding"
)

func TestInternalUpdateFeedUsesPackagedStableChannel(t *testing.T) {
	originalProfile := branding.PackagedProfile
	originalServer := packagedEnterpriseServerURL
	originalFeed := packagedDesktopUpdateFeedURL
	branding.PackagedProfile = "internal"
	packagedEnterpriseServerURL = "http://192.168.9.177:8080"
	packagedDesktopUpdateFeedURL = "http://192.168.9.177:8081/downloads/channels/stable.json"
	t.Setenv("XELORA_UPDATE_FEED_URL", "")
	t.Cleanup(func() {
		branding.PackagedProfile = originalProfile
		packagedEnterpriseServerURL = originalServer
		packagedDesktopUpdateFeedURL = originalFeed
	})

	if got := desktopUpdateFeedURL(); got != "http://192.168.9.177:8081/downloads/channels/stable.json" {
		t.Fatalf("internal update feed = %q, want packaged stable channel", got)
	}
}

func TestPublicUpdateFeedRemainsGitHub(t *testing.T) {
	originalProfile := branding.PackagedProfile
	originalFeed := packagedDesktopUpdateFeedURL
	branding.PackagedProfile = "public"
	packagedDesktopUpdateFeedURL = ""
	t.Setenv("XELORA_UPDATE_FEED_URL", "")
	t.Cleanup(func() {
		branding.PackagedProfile = originalProfile
		packagedDesktopUpdateFeedURL = originalFeed
	})

	if got := desktopUpdateFeedURL(); got != "https://api.github.com/repos/Tencent/Xelora/releases/latest" {
		t.Fatalf("public update feed = %q, want GitHub feed", got)
	}
}

func TestParseInternalReleaseManifestValidatesReleaseIntegrityMetadata(t *testing.T) {
	manifest, err := parseInternalReleaseManifest([]byte(`{
        "schemaVersion": 1,
        "product": "Knavis Personal",
        "channel": "stable",
        "latest": {
            "version": "1.0.0",
            "file": "Knavis-Personal-1.0.0.zip",
            "sha256": "caf4cfc6ad3a6c3538af2ef3590aa34ef293738eb5bb0ff65ae6e1a6c0b43020",
            "sizeBytes": 88875283
        }
    }`))
	if err != nil {
		t.Fatalf("parse valid manifest: %v", err)
	}
	if manifest.Latest.Version != "1.0.0" {
		t.Fatalf("latest version = %q, want 1.0.0", manifest.Latest.Version)
	}

	for name, body := range map[string]string{
		"unsafe filename":  `{"schemaVersion":1,"product":"Knavis Personal","channel":"stable","latest":{"version":"1.0.0","file":"../client.zip","sha256":"caf4cfc6ad3a6c3538af2ef3590aa34ef293738eb5bb0ff65ae6e1a6c0b43020","sizeBytes":1}}`,
		"invalid checksum": `{"schemaVersion":1,"product":"Knavis Personal","channel":"stable","latest":{"version":"1.0.0","file":"client.zip","sha256":"bad","sizeBytes":1}}`,
		"missing size":     `{"schemaVersion":1,"product":"Knavis Personal","channel":"stable","latest":{"version":"1.0.0","file":"client.zip","sha256":"caf4cfc6ad3a6c3538af2ef3590aa34ef293738eb5bb0ff65ae6e1a6c0b43020","sizeBytes":0}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseInternalReleaseManifest([]byte(body)); err == nil {
				t.Fatal("expected manifest validation error")
			}
		})
	}
}

func TestInternalReleaseDownloadURLStaysOnServerDownloadsPath(t *testing.T) {
	got, err := internalReleaseDownloadURL(
		"http://192.168.9.177:8081/downloads/channels/stable.json",
		"Knavis-Personal-1.0.0.zip",
	)
	if err != nil {
		t.Fatalf("build download URL: %v", err)
	}
	if got != "http://192.168.9.177:8081/downloads/Knavis-Personal-1.0.0.zip" {
		t.Fatalf("download URL = %q, want same-origin downloads path", got)
	}
}
