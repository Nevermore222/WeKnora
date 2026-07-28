package types

import "testing"

func TestUserHasCapability(t *testing.T) {
	u := &User{
		CanCreateKnowledgeBase: false,
		CanCreateAgent:         true,
		CanBindModel:           false,
	}
	if UserHasCapability(u, CapabilityCreateKnowledgeBase) {
		t.Fatal("expected create KB denied")
	}
	if !UserHasCapability(u, CapabilityCreateAgent) {
		t.Fatal("expected create agent allowed")
	}
	if UserHasCapability(nil, CapabilityCreateAgent) {
		t.Fatal("nil user must deny")
	}
	if UserHasCapability(u, "unknown") {
		t.Fatal("unknown capability must deny")
	}
}

func TestCapabilityRegistry(t *testing.T) {
	regs := CapabilityRegistry()
	if len(regs) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(regs))
	}
}
