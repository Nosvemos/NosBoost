package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"testing"
)

func TestJSONDataIntegrity(t *testing.T) {
	// Create a complex mock baseline system state
	mockState := SystemBaselineState{
		Version:   "1.0.0",
		Timestamp: "2026-05-24T15:00:00Z",
		Devices: []DeviceBackupState{
			{
				DevicePath:           `PCI\VEN_10DE&DEV_2484&SUBSYS_38921462&REV_A1\3&11583659&0&00`,
				MSISupportedExists:   true,
				MSISupportedValue:    1,
				DevicePriorityExists: true,
				DevicePriorityValue:  3,
			},
		},
		Network: NetworkBackupState{
			NICs: []NICBackupState{
				{
					InterfaceGUID:         "{12345678-ABCD-EF01-2345-6789ABCDEF01}",
					TcpAckFrequencyExists: true,
					TcpAckFrequencyValue:  1,
					TCPNoDelayExists:      true,
					TCPNoDelayValue:       1,
				},
			},
			NetworkThrottlingExists:    true,
			NetworkThrottlingValue:     10,
			SystemResponsivenessExists: true,
			SystemResponsivenessValue:  0,
		},
		Power: PowerBackupState{
			OriginalActiveScheme: "381b4222-f694-41f0-9685-ff5bb260df2e",
		},
		Services: []ServiceBackupState{
			{
				ServiceName: "SysMain",
				StartExists: true,
				StartValue:  2,
			},
		},
	}

	// Marshal mock state
	bytes, err := json.Marshal(mockState)
	if err != nil {
		t.Fatalf("failed to marshal mock baseline state: %v", err)
	}

	// Unmarshal mock state
	var unmarshaledState SystemBaselineState
	if err := json.Unmarshal(bytes, &unmarshaledState); err != nil {
		t.Fatalf("failed to unmarshal baseline state bytes: %v", err)
	}

	// Assert version matches
	if unmarshaledState.Version != mockState.Version {
		t.Errorf("version mismatch: expected %q, got %q", mockState.Version, unmarshaledState.Version)
	}

	// Assert active power scheme matches
	if unmarshaledState.Power.OriginalActiveScheme != mockState.Power.OriginalActiveScheme {
		t.Errorf("power scheme mismatch: expected %q, got %q", mockState.Power.OriginalActiveScheme, unmarshaledState.Power.OriginalActiveScheme)
	}

	// Assert hardware parameters match
	if len(unmarshaledState.Devices) != 1 || unmarshaledState.Devices[0].DevicePath != mockState.Devices[0].DevicePath {
		t.Errorf("hardware device parameters mismatch")
	}

	if unmarshaledState.Devices[0].DevicePriorityValue != mockState.Devices[0].DevicePriorityValue {
		t.Errorf("device priority mismatch: expected %d, got %d", mockState.Devices[0].DevicePriorityValue, unmarshaledState.Devices[0].DevicePriorityValue)
	}
}

func TestLoadMissingBaselineFile(t *testing.T) {
	// Temporarily remove state backup file if it exists
	originalExists := true
	if _, err := os.Stat(BackupFileName); errors.Is(err, fs.ErrNotExist) {
		originalExists = false
	}

	if originalExists {
		err := os.Rename(BackupFileName, BackupFileName+".tmp")
		if err != nil {
			t.Fatalf("failed to temporarily rename backup file: %v", err)
		}
		defer func() {
			_ = os.Rename(BackupFileName+".tmp", BackupFileName)
		}()
	}

	// Attempt loading missing file
	_, err := LoadBaselineState()
	if err == nil {
		t.Error("expected error loading non-existent state backup file, got nil")
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	// Skip test if not running in Administrator mode (since SaveBaselineState accesses registry)
	if !isAdmin() {
		t.Skip("skipping Windows registry live roundtrip test; not running in Administrator shell")
	}

	// Backup existing baseline state file if it exists
	originalExists := true
	if _, err := os.Stat(BackupFileName); errors.Is(err, fs.ErrNotExist) {
		originalExists = false
	}

	if originalExists {
		err := os.Rename(BackupFileName, BackupFileName+".backup")
		if err != nil {
			t.Fatalf("failed to backup existing baseline: %v", err)
		}
		defer func() {
			_ = os.Remove(BackupFileName) // Clean up the test result
			_ = os.Rename(BackupFileName+".backup", BackupFileName)
		}()
	} else {
		defer func() {
			_ = os.Remove(BackupFileName) // Clean up the test result
		}()
	}

	// 1. Save system state
	saved, err := SaveBaselineState()
	if err != nil {
		t.Fatalf("SaveBaselineState failed: %v", err)
	}

	// 2. Load system state
	loaded, err := LoadBaselineState()
	if err != nil {
		t.Fatalf("LoadBaselineState failed: %v", err)
	}

	// 3. Verify loaded matches saved
	if loaded.Timestamp != saved.Timestamp {
		t.Errorf("timestamp mismatch: expected %q, got %q", saved.Timestamp, loaded.Timestamp)
	}

	if loaded.Power.OriginalActiveScheme != saved.Power.OriginalActiveScheme {
		t.Errorf("power scheme mismatch: expected %q, got %q", saved.Power.OriginalActiveScheme, loaded.Power.OriginalActiveScheme)
	}
}
