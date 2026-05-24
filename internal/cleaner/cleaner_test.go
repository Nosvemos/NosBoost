package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanerConstants(t *testing.T) {
	if ProcessIoPriority != 33 {
		t.Errorf("ProcessIoPriority class index mismatch")
	}

	if IoPriorityVeryLow != 0 {
		t.Errorf("IoPriorityVeryLow value mismatch")
	}

	if IoPriorityLow != 1 {
		t.Errorf("IoPriorityLow value mismatch")
	}

	if IoPriorityNormal != 2 {
		t.Errorf("IoPriorityNormal value mismatch")
	}
}

func TestNTDLLInformationProcessWrappers(t *testing.T) {
	// Verify that the NTDLL dynamic procedure is loaded cleanly
	if procNtSetInformationProcess == nil {
		t.Errorf("failed to map ntdll procedure NtSetInformationProcess")
	}
}

func TestSafeWipeDirectoryOperations(t *testing.T) {
	// Create a temporary mock directory
	tempDir, err := os.MkdirTemp("", "nosboost_cleaner_test")
	if err != nil {
		t.Fatalf("failed to create temp test directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Final cleanup

	// Write mock junk files
	file1 := filepath.Join(tempDir, "junk1.tmp")
	err = os.WriteFile(file1, []byte("junk content 1"), 0644)
	if err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	file2 := filepath.Join(tempDir, "junk2.log")
	err = os.WriteFile(file2, []byte("log content"), 0644)
	if err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	metrics := &CleanupMetrics{}
	
	// Test the wipeDirectory traversal
	wipeDirectory(tempDir, metrics)

	if metrics.FilesDeleted != 2 {
		t.Errorf("expected 2 files deleted, got %d", metrics.FilesDeleted)
	}

	if metrics.BytesFreed <= 0 {
		t.Errorf("expected bytes freed to be greater than 0, got %d", metrics.BytesFreed)
	}
}

func TestResolveEnvironmentPaths(t *testing.T) {
	temp := os.Getenv("TEMP")
	if temp == "" {
		t.Skip("skipping TEMP env check; environment variable not set")
	}

	// Verify that the path resolved matches basic drive structure
	if !filepath.IsAbs(temp) {
		t.Errorf("resolved TEMP path %q is not absolute", temp)
	}
}
