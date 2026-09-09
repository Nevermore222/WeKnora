package types

import "testing"

func TestClassificationTagsNilValueAndScanNormalizeToEmptyObject(t *testing.T) {
	var tags ClassificationTags

	value, err := tags.Value()
	if err != nil {
		t.Fatalf("Value() returned error: %v", err)
	}
	bytes, ok := value.([]byte)
	if !ok || string(bytes) != "{}" {
		t.Fatalf("Value() = %#v, want []byte(\"{}\")", value)
	}

	if err := tags.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) returned error: %v", err)
	}
	if tags == nil {
		t.Fatal("Scan(nil) left tags nil")
	}
}

func TestCustomAgentEnsureDefaultsInitializesClassificationTags(t *testing.T) {
	agent := &CustomAgent{}
	agent.EnsureDefaults()
	if agent.ClassificationTags == nil {
		t.Fatal("EnsureDefaults left ClassificationTags nil")
	}
}
