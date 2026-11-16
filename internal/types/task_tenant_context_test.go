package types

import "testing"

func TestKBClonePayload_TenantFallbacks(t *testing.T) {
	t.Run("explicit source and target win", func(t *testing.T) {
		p := KBClonePayload{
			SourceTenantID: 10000,
			TargetTenantID: 10011,
			TenantID:       99999,
		}
		if got := p.SourceTenant(); got != 10000 {
			t.Fatalf("SourceTenant() = %d, want 10000", got)
		}
		if got := p.TargetTenant(); got != 10011 {
			t.Fatalf("TargetTenant() = %d, want 10011", got)
		}
	})

	t.Run("legacy tenant applies to source and target", func(t *testing.T) {
		p := KBClonePayload{TenantID: 10000}
		if got := p.SourceTenant(); got != 10000 {
			t.Fatalf("SourceTenant() = %d, want 10000", got)
		}
		if got := p.TargetTenant(); got != 10000 {
			t.Fatalf("TargetTenant() = %d, want 10000", got)
		}
	})
}

func TestKBDeletePayload_SourceTenantFallback(t *testing.T) {
	if got := (KBDeletePayload{SourceTenantID: 10000, TenantID: 99999}).SourceTenant(); got != 10000 {
		t.Fatalf("explicit SourceTenant() = %d, want 10000", got)
	}
	if got := (KBDeletePayload{TenantID: 10000}).SourceTenant(); got != 10000 {
		t.Fatalf("legacy SourceTenant() = %d, want 10000", got)
	}
}

func TestKnowledgeMovePayload_TenantFallbacks(t *testing.T) {
	t.Run("explicit source and target win", func(t *testing.T) {
		p := KnowledgeMovePayload{
			SourceTenantID: 10000,
			TargetTenantID: 10011,
			TenantID:       99999,
		}
		if got := p.SourceTenant(); got != 10000 {
			t.Fatalf("SourceTenant() = %d, want 10000", got)
		}
		if got := p.TargetTenant(); got != 10011 {
			t.Fatalf("TargetTenant() = %d, want 10011", got)
		}
	})

	t.Run("legacy tenant applies to source and target", func(t *testing.T) {
		p := KnowledgeMovePayload{TenantID: 10000}
		if got := p.SourceTenant(); got != 10000 {
			t.Fatalf("SourceTenant() = %d, want 10000", got)
		}
		if got := p.TargetTenant(); got != 10000 {
			t.Fatalf("TargetTenant() = %d, want 10000", got)
		}
	})
}
