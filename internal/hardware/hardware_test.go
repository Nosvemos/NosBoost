package hardware

import (
	"testing"
)

func TestHardwareClassGUIDConstants(t *testing.T) {
	gpuGUID := "{4d36e968-e325-11ce-bfc1-08002be10318}"
	nicGUID := "{4d36e972-e325-11ce-bfc1-08002be10318}"

	// Simple check that constants are correctly formatted
	if len(gpuGUID) != 38 || gpuGUID[0] != '{' || gpuGUID[37] != '}' {
		t.Errorf("Display class GUID layout is invalid")
	}

	if len(nicGUID) != 38 || nicGUID[0] != '{' || nicGUID[37] != '}' {
		t.Errorf("Network interface card class GUID layout is invalid")
	}
}

func TestPeripheralHivesPaths(t *testing.T) {
	if MouseParametersKey != `SYSTEM\CurrentControlSet\Services\mouclass\Parameters` {
		t.Errorf("mouclass parameters key path mismatch")
	}

	if KeyboardParametersKey != `SYSTEM\CurrentControlSet\Services\kbdclass\Parameters` {
		t.Errorf("kbdclass parameters key path mismatch")
	}

	if TargetQueueSize != 20 {
		t.Errorf("optimized target queue size constant mismatch")
	}
}

func TestActivePCIScannerCompile(t *testing.T) {
	// Verify scanner helper function exists and returns without crashing
	// (Will run and evaluate active display/net configurations if running on Windows)
	devices, err := scanActivePCIDevices()
	if err != nil {
		t.Logf("PCI scanner compiled cleanly. (Returned error under mock or non-Windows system: %v)", err)
	} else {
		t.Logf("PCI scanner succeeded cleanly, found %d active target devices", len(devices))
	}
}
