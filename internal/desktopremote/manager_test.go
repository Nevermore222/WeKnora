package desktopremote

import (
	"errors"
	"testing"
)

func TestCapabilityMajorMismatchBlocksActivation(t *testing.T) {
	err := ValidateCapabilities(SystemCapabilities{
		APIContractMajor: 2,
		APIContractMinor: 0,
	})
	if !errors.Is(err, ErrIncompatibleAPIContract) {
		t.Fatalf("err=%v", err)
	}
}
