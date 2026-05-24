package syswatch

import (
	"testing"
)

func TestSyswatchPrivilegeAndHandleConstants(t *testing.T) {
	if PROCESS_SUSPEND_RESUME != 0x0800 {
		t.Errorf("PROCESS_SUSPEND_RESUME constant mismatch")
	}

	if SeDebugPrivilege != "SeDebugPrivilege" {
		t.Errorf("SeDebugPrivilege string value mismatch")
	}
}

func TestNTDLLSuspenderWrappers(t *testing.T) {
	// Verify that the NTDLL dynamic suspend/resume procedures are loaded cleanly
	if procNtSuspendProcess == nil {
		t.Errorf("failed to map ntdll procedure NtSuspendProcess")
	}

	if procNtResumeProcess == nil {
		t.Errorf("failed to map ntdll procedure NtResumeProcess")
	}
}

func TestGraphicsRegistryPaths(t *testing.T) {
	if GraphicsDriversKey != `SYSTEM\CurrentControlSet\Control\GraphicsDrivers` {
		t.Errorf("graphics drivers key path mismatch")
	}

	if GameBarKey != `Software\Microsoft\GameBar` {
		t.Errorf("game bar key path mismatch")
	}
}

func TestGetHAGSCompile(t *testing.T) {
	// Verify HAGS check compiles and runs cleanly, logging state
	state, err := GetHAGSState()
	if err != nil {
		t.Logf("HAGS query completed. (Error on unsupported systems: %v)", err)
	} else {
		t.Logf("HAGS status is: %d (2=Active, 1=Disabled, 0=Not Present)", state)
	}
}

func TestGetGameModeCompile(t *testing.T) {
	// Verify Game Mode check compiles and runs cleanly, logging state
	state, err := GetGameModeState()
	if err != nil {
		t.Logf("Game Mode query completed. (Error: %v)", err)
	} else {
		t.Logf("Game Mode active is: %t", state)
	}
}
