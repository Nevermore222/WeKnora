package service

import "testing"

func TestNormalizeInviteCodeTrimsCopiedWhitespace(t *testing.T) {
	if got := normalizeInviteCode("  6784ab31378e2d6e\n"); got != "6784ab31378e2d6e" {
		t.Fatalf("normalized code=%q", got)
	}
}

func TestNormalizeInviteCodeKeepsCodeCase(t *testing.T) {
	if got := normalizeInviteCode(" AbCd1234 "); got != "AbCd1234" {
		t.Fatalf("normalized code=%q", got)
	}
}
