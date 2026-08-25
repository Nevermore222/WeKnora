package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Tencent/Xelora/internal/application/service"
)

func TestInviteCodeAppErrorExplainsExpiredCode(t *testing.T) {
	appErr := inviteCodeAppError(service.ErrInviteCodeExpired)
	if appErr.HTTPCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", appErr.HTTPCode, http.StatusBadRequest)
	}
	if appErr.Message != "Invite code has expired; ask the organization administrator to refresh it." {
		t.Fatalf("message=%q", appErr.Message)
	}
}

func TestInviteCodeAppErrorHidesUnknownCodeDetails(t *testing.T) {
	appErr := inviteCodeAppError(errors.New("database details"))
	if appErr.HTTPCode != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", appErr.HTTPCode, http.StatusNotFound)
	}
	if appErr.Message != "Invalid invite code" {
		t.Fatalf("message=%q", appErr.Message)
	}
}
