package service

import "testing"

func TestClientWorkspaceBindingSettingDefaultsOff(t *testing.T) {
	item, ok := registry["client.workspace_binding_enabled"]
	if !ok {
		t.Fatal("missing client workspace binding setting")
	}
	if item.Type != "bool" {
		t.Fatalf("type=%q, want bool", item.Type)
	}
	if item.Default != false {
		t.Fatalf("default=%v, want false", item.Default)
	}
}
