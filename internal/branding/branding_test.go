package branding

import "testing"

func TestPublicBrandingIsTheDefault(t *testing.T) {
	original := PackagedProfile
	PackagedProfile = ""
	t.Cleanup(func() { PackagedProfile = original })

	if got := BrandName(); got != "Xelora" {
		t.Fatalf("brand name = %q, want Xelora", got)
	}
	if got := ProductName(); got != "Xelora Personal" {
		t.Fatalf("product name = %q, want Xelora Personal", got)
	}
}

func TestInternalBrandingUsesKnavisEverywhere(t *testing.T) {
	original := PackagedProfile
	PackagedProfile = "internal"
	t.Cleanup(func() { PackagedProfile = original })

	if got := BrandName(); got != "Knavis" {
		t.Fatalf("brand name = %q, want Knavis", got)
	}
	if got := ProductName(); got != "Knavis Personal" {
		t.Fatalf("product name = %q, want Knavis Personal", got)
	}
}
