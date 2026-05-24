package booster

import (
	"testing"
)

func TestAffinityBitmaskManipulation(t *testing.T) {
	// Let's mock a standard 8-core CPU (system affinity mask = 0xFF / 255)
	sysAff := uintptr(0xFF)

	// Isolate game mask (cores 2 to 7): should clear Core 0 and Core 1 (bits 0 and 1)
	gameMask := sysAff &^ uintptr(3)

	expectedMask := uintptr(0xFC) // 252 in decimal
	if gameMask != expectedMask {
		t.Errorf("affinity bitmask slice failed: expected 0x%X (252), got 0x%X (%d)", expectedMask, gameMask, gameMask)
	}

	// Verify dual core safeguard limits (sysAff <= 3)
	// Dual core system affinity = 3 (0x03)
	dualSysAff := uintptr(3)
	gameMaskDual := dualSysAff &^ uintptr(3)
	if gameMaskDual != 0 {
		t.Errorf("dual core mask bit extraction anomaly")
	}
}

func TestProcessorSubgroups(t *testing.T) {
	if ProcessorSubgroup != "54533251-82be-4824-96c1-47b60b740d00" {
		t.Errorf("processor subgroup GUID mismatch")
	}
	if MinCoresSetting != "0cc5b647-c1df-4615-815a-8deb02312a2c" {
		t.Errorf("min cores GUID mismatch")
	}
	if MaxCoresSetting != "ea062031-0e34-4ff1-9b6d-eb1059334028" {
		t.Errorf("max cores GUID mismatch")
	}
}

func TestBackgroundTargetsList(t *testing.T) {
	if len(BackgroundTargets) == 0 {
		t.Errorf("background targets list is empty")
	}

	foundDiscord := false
	for _, target := range BackgroundTargets {
		if target == "discord.exe" {
			foundDiscord = true
			break
		}
	}
	if !foundDiscord {
		t.Errorf("expected discord.exe to be in background targets list")
	}
}
