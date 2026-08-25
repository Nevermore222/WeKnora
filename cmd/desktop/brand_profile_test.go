package main

import (
	"testing"

	"github.com/Tencent/Xelora/internal/branding"
)

func TestDesktopBrandProfileDefaultsToPublic(t *testing.T) {
	original := branding.PackagedProfile
	branding.PackagedProfile = ""
	t.Setenv("XELORA_PERSONAL_SERVER_NAME", "")
	t.Cleanup(func() { branding.PackagedProfile = original })

	if got := desktopBrandName(); got != "Xelora" {
		t.Fatalf("desktop brand name = %q, want Xelora", got)
	}
	if got := desktopProductName(); got != "Xelora Personal" {
		t.Fatalf("desktop product name = %q, want Xelora Personal", got)
	}
	if got := defaultEnterpriseServerName(); got != "Xelora Server" {
		t.Fatalf("default server name = %q, want Xelora Server", got)
	}
}

func TestDesktopBrandProfileUsesKnavisForInternalBuild(t *testing.T) {
	original := branding.PackagedProfile
	branding.PackagedProfile = "internal"
	t.Setenv("XELORA_PERSONAL_SERVER_NAME", "")
	t.Cleanup(func() { branding.PackagedProfile = original })

	if got := desktopBrandName(); got != "Knavis" {
		t.Fatalf("desktop brand name = %q, want Knavis", got)
	}
	if got := desktopProductName(); got != "Knavis Personal" {
		t.Fatalf("desktop product name = %q, want Knavis Personal", got)
	}
	if got := defaultEnterpriseServerName(); got != "Knavis Server" {
		t.Fatalf("default server name = %q, want Knavis Server", got)
	}
}

func TestDesktopUpdateFeedUsesInternalStableChannelAndKeepsPublicFeed(t *testing.T) {
	original := branding.PackagedProfile
	originalServer := packagedEnterpriseServerURL
	originalFeed := packagedDesktopUpdateFeedURL
	branding.PackagedProfile = "internal"
	packagedEnterpriseServerURL = "http://192.168.9.177:8080"
	packagedDesktopUpdateFeedURL = "http://192.168.9.177:8081/downloads/channels/stable.json"
	t.Setenv("XELORA_UPDATE_FEED_URL", "")
	t.Cleanup(func() {
		branding.PackagedProfile = original
		packagedEnterpriseServerURL = originalServer
		packagedDesktopUpdateFeedURL = originalFeed
	})

	if got := desktopUpdateFeedURL(); got != packagedDesktopUpdateFeedURL {
		t.Fatalf("internal default update feed = %q, want packaged stable channel", got)
	}

	branding.PackagedProfile = "public"
	if got := desktopUpdateFeedURL(); got != "https://api.github.com/repos/Tencent/Xelora/releases/latest" {
		t.Fatalf("public default update feed = %q, want upstream Xelora feed", got)
	}
}
