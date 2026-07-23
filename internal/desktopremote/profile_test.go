package desktopremote

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeServerOrigin(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		want          string
		allowInsecure bool
		wantErr       bool
	}{
		{name: "https", raw: "https://xelora.example.com/", want: "https://xelora.example.com"},
		{name: "loopback http", raw: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080"},
		{name: "localhost http", raw: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "explicit insecure lan", raw: "http://192.168.1.10:8080", want: "http://192.168.1.10:8080", allowInsecure: true},
		{name: "reject insecure lan", raw: "http://192.168.1.10:8080", wantErr: true},
		{name: "reject credentials", raw: "https://user:pass@example.com", wantErr: true},
		{name: "reject path", raw: "https://example.com/xelora", wantErr: true},
		{name: "reject file", raw: "file:///c:/data", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeServerOrigin(tt.raw, tt.allowInsecure)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeServerOrigin(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeServerOrigin(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestResolveTargetCannotChangeOrigin(t *testing.T) {
	profile := ServerProfile{BaseURL: "https://xelora.example.com"}

	if _, err := profile.ResolveTarget("//attacker.example/api/v1"); err == nil {
		t.Fatal("scheme-relative target must be rejected")
	}
	if _, err := profile.ResolveTarget("https://attacker.example/api/v1"); err == nil {
		t.Fatal("absolute target must be rejected")
	}

	got, err := profile.ResolveTarget("/api/v1/sessions?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "https://xelora.example.com/api/v1/sessions?limit=10" {
		t.Fatalf("unexpected target %q", got.String())
	}
}

func TestProfileStoreCRUDNormalizesURLAndKeepsIDImmutable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:desktopremote-profile?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	store := NewProfileStore(db)
	if err := store.AutoMigrate(); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	first, err := store.Create(ctx, ServerProfile{Name: "Primary", BaseURL: "https://one.example/"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(ctx, ServerProfile{Name: "Local", BaseURL: "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Fatalf("invalid generated IDs %q %q", first.ID, second.ID)
	}
	if first.BaseURL != "https://one.example" {
		t.Fatalf("URL was not normalized: %q", first.BaseURL)
	}

	profiles, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 || profiles[0].ID != first.ID || profiles[1].ID != second.ID {
		t.Fatalf("profiles not returned in creation order: %+v", profiles)
	}

	updated, err := store.Update(ctx, first.ID, ServerProfile{
		ID:      "replacement-id",
		Name:    "Primary renamed",
		BaseURL: "https://two.example/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != first.ID || updated.Name != "Primary renamed" || updated.BaseURL != "https://two.example" {
		t.Fatalf("unexpected update: %+v", updated)
	}

	if err := store.Delete(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	profiles, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].ID != first.ID {
		t.Fatalf("unexpected profiles after delete: %+v", profiles)
	}
}
