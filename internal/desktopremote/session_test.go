package desktopremote

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestSessionSnapshotNeverContainsTokens(t *testing.T) {
	sessions := NewSessionStore()
	sessions.Set("server-a", &Session{
		UserID:            "u1",
		AccessToken:       "access",
		RefreshTokenOwner: "u1",
		AccessExpiresAt:   time.Now().Add(time.Hour),
		Snapshot: IdentitySnapshot{
			Authenticated: true,
			UserID:        "u1",
			User:          json.RawMessage(`{"id":"u1","email":"user@example.com"}`),
		},
	})

	raw, err := json.Marshal(sessions.Snapshot("server-a"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("access")) {
		t.Fatalf("snapshot leaked token: %s", raw)
	}
	if sessions.Snapshot("server-b").Authenticated {
		t.Fatal("sessions must be scoped by profile")
	}
}

func TestSessionStoreReturnsCopies(t *testing.T) {
	sessions := NewSessionStore()
	sessions.Set("server-a", &Session{
		UserID:            "u1",
		AccessToken:       "access",
		RefreshTokenOwner: "u1",
		Snapshot:          IdentitySnapshot{Authenticated: true, UserID: "u1"},
	})

	got, ok := sessions.Get("server-a")
	if !ok {
		t.Fatal("session missing")
	}
	got.AccessToken = "mutated"

	again, _ := sessions.Get("server-a")
	if again.AccessToken != "access" {
		t.Fatalf("session store leaked mutable state: %q", again.AccessToken)
	}
}
