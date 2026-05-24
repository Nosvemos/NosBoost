package memory

import (
	"os"
	"testing"
)

func TestTrimmerExclusions(t *testing.T) {
	myPID := uint32(os.Getpid())

	// Test that critical system processes are mapped correctly
	if !SystemCriticalProcesses["svchost.exe"] {
		t.Errorf("critical process map check failed: svchost.exe not flagged as critical")
	}

	if !SystemCriticalProcesses["lsass.exe"] {
		t.Errorf("critical process map check failed: lsass.exe not flagged as critical")
	}

	// Verify that a user process like "discord.exe" is NOT in critical processes exclusion list
	if SystemCriticalProcesses["discord.exe"] {
		t.Errorf("discord.exe was incorrectly flagged as critical system process")
	}

	// Verify self PID is handled
	if myPID == 0 {
		t.Errorf("invalid self PID")
	}
}

func TestNTDLLPurgerWrappers(t *testing.T) {
	if SystemMemoryListInformation != 80 {
		t.Errorf("undocumented NT class constant mismatch")
	}
	if MemoryPurgeStandbyList != 4 {
		t.Errorf("purge standby command enum mismatch")
	}
	if MemoryFlushModifiedList != 5 {
		t.Errorf("flush modified command enum mismatch")
	}

	// Verify that the NTDLL dynamic procedure is loaded
	if procNtSetSystemInformation == nil {
		t.Errorf("failed to map ntdll procedure NtSetSystemInformation")
	}
}

func TestSafeRecoveryBlock(t *testing.T) {
	// Execute a call with intentionally invalid parameters (nil pointer and 0 length)
	// representing an access violation to verify the safe recover defer block catches it.
	_, err := ntSetSystemInformation(999, nil, 0)
	
	// It should either return an NT status error or recover smoothly without crashing the Go runtime.
	if err == nil {
		// If it actually succeeded with invalid parameters on some platform, it should not panic.
		t.Log("invalid NT call executed without panic or error")
	} else {
		t.Logf("graceful failure logged: %v", err)
	}
}

func TestSePrivilegesConstants(t *testing.T) {
	if SeProfileSingleProcessPrivilege != "SeProfileSingleProcessPrivilege" {
		t.Errorf("privilege SeProfileSingleProcessPrivilege string mismatch")
	}
	if SeIncreaseQuotaPrivilege != "SeIncreaseQuotaPrivilege" {
		t.Errorf("privilege SeIncreaseQuotaPrivilege string mismatch")
	}
}
