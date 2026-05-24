package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"nosboost/internal/system"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// isAdmin checks elevated state before running registry/service operations.
func isAdmin() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

// scanPCIDevices discovers active devices under the PCI subkey tree.
func scanPCIDevices() ([]string, error) {
	pciKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Enum\PCI`, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open PCI registry tree: %w", err)
	}
	defer pciKey.Close()

	vendors, err := pciKey.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read PCI subkeys: %w", err)
	}

	var devices []string
	for _, vendor := range vendors {
		vendorKeyPath := `SYSTEM\CurrentControlSet\Enum\PCI\` + vendor
		vendorKey, err := registry.OpenKey(registry.LOCAL_MACHINE, vendorKeyPath, registry.READ)
		if err != nil {
			continue
		}
		instances, err := vendorKey.ReadSubKeyNames(-1)
		vendorKey.Close()
		if err != nil {
			continue
		}

		for _, instance := range instances {
			devices = append(devices, vendor+`\`+instance)
		}
	}
	return devices, nil
}

// SaveBaselineState scans the OS configuration and serializes the baseline snapshot to disk.
func SaveBaselineState() (*SystemBaselineState, error) {
	if !isAdmin() {
		return nil, errors.New("administrative privileges required to back up system baseline state")
	}

	baseline := &SystemBaselineState{
		Version:   "1.0.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Devices:   []DeviceBackupState{},
		Services:  []ServiceBackupState{},
	}

	// 1. BACK UP HARDWARE INTERRUPT STATE (PCI GPU & NIC)
	devicePaths, err := scanPCIDevices()
	if err == nil {
		for _, devPath := range devicePaths {
			fullPath := `SYSTEM\CurrentControlSet\Enum\PCI\` + devPath
			key, err := registry.OpenKey(registry.LOCAL_MACHINE, fullPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			classGUID, _, err := key.GetStringValue("ClassGUID")
			key.Close()
			if err != nil {
				continue
			}

			// Display Class: {4d36e968-e325-11ce-bfc1-08002be10318}
			// Network Class: {4d36e972-e325-11ce-bfc1-08002be10318}
			if classGUID == "{4d36e968-e325-11ce-bfc1-08002be10318}" || classGUID == "{4d36e972-e325-11ce-bfc1-08002be10318}" {
				devBackup := DeviceBackupState{DevicePath: devPath}

				// Check MSISupported
				msiPath := fullPath + `\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties`
				if msiKey, err := registry.OpenKey(registry.LOCAL_MACHINE, msiPath, registry.QUERY_VALUE); err == nil {
					if val, _, err := msiKey.GetIntegerValue("MSISupported"); err == nil {
						devBackup.MSISupportedExists = true
						devBackup.MSISupportedValue = uint32(val)
					}
					msiKey.Close()
				}

				// Check DevicePriority
				affPath := fullPath + `\Device Parameters\Interrupt Management\Affinity Policy`
				if affKey, err := registry.OpenKey(registry.LOCAL_MACHINE, affPath, registry.QUERY_VALUE); err == nil {
					if val, _, err := affKey.GetIntegerValue("DevicePriority"); err == nil {
						devBackup.DevicePriorityExists = true
						devBackup.DevicePriorityValue = uint32(val)
					}
					affKey.Close()
				}

				baseline.Devices = append(baseline.Devices, devBackup)
			}
		}
	}

	// 2. BACK UP SYSTEM LATENCY REGISTRY SETTINGS
	sysProfilePath := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	if sysKey, err := registry.OpenKey(registry.LOCAL_MACHINE, sysProfilePath, registry.QUERY_VALUE); err == nil {
		defer sysKey.Close()
		if val, _, err := sysKey.GetIntegerValue("NetworkThrottlingIndex"); err == nil {
			baseline.Network.NetworkThrottlingExists = true
			baseline.Network.NetworkThrottlingValue = uint32(val)
		}
		if val, _, err := sysKey.GetIntegerValue("SystemResponsiveness"); err == nil {
			baseline.Network.SystemResponsivenessExists = true
			baseline.Network.SystemResponsivenessValue = uint32(val)
		}
	}

	// 3. BACK UP NETWORK ADAPTER LATENCY OVERRIDES (TcpAckFrequency & TCPNoDelay)
	nicRootPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
	if rootKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicRootPath, registry.READ); err == nil {
		if guids, err := rootKey.ReadSubKeyNames(-1); err == nil {
			for _, guid := range guids {
				nicPath := nicRootPath + `\` + guid
				if nicKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicPath, registry.QUERY_VALUE); err == nil {
					nicBackup := NICBackupState{InterfaceGUID: guid}
					if val, _, err := nicKey.GetIntegerValue("TcpAckFrequency"); err == nil {
						nicBackup.TcpAckFrequencyExists = true
						nicBackup.TcpAckFrequencyValue = uint32(val)
					}
					if val, _, err := nicKey.GetIntegerValue("TCPNoDelay"); err == nil {
						nicBackup.TCPNoDelayExists = true
						nicBackup.TCPNoDelayValue = uint32(val)
					}
					baseline.Network.NICs = append(baseline.Network.NICs, nicBackup)
					nicKey.Close()
				}
			}
		}
		rootKey.Close()
	}

	// 4. BACK UP ACTIVE POWER SCHEME GUID & CORE PARKING SETTINGS
	powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`
	if powerKey, err := registry.OpenKey(registry.LOCAL_MACHINE, powerPath, registry.QUERY_VALUE); err == nil {
		if scheme, _, err := powerKey.GetStringValue("ActivePowerScheme"); err == nil {
			baseline.Power.OriginalActiveScheme = scheme

			// Min Cores path
			minCoresPath := fmt.Sprintf(`%s\%s\54533251-82be-4824-96c1-47b60b740d00\0cc5b647-c1df-4615-815a-8deb02312a2c`, powerPath, scheme)
			if mcKey, err := registry.OpenKey(registry.LOCAL_MACHINE, minCoresPath, registry.QUERY_VALUE); err == nil {
				if val, _, err := mcKey.GetIntegerValue("ACSettingIndex"); err == nil {
					baseline.Power.MinCoresACExists = true
					baseline.Power.MinCoresACValue = uint32(val)
				}
				if val, _, err := mcKey.GetIntegerValue("DCSettingIndex"); err == nil {
					baseline.Power.MinCoresDCExists = true
					baseline.Power.MinCoresDCValue = uint32(val)
				}
				mcKey.Close()
			}

			// Max Cores path
			maxCoresPath := fmt.Sprintf(`%s\%s\54533251-82be-4824-96c1-47b60b740d00\ea062031-0e34-4ff1-9b6d-eb1059334028`, powerPath, scheme)
			if xcKey, err := registry.OpenKey(registry.LOCAL_MACHINE, maxCoresPath, registry.QUERY_VALUE); err == nil {
				if val, _, err := xcKey.GetIntegerValue("ACSettingIndex"); err == nil {
					baseline.Power.MaxCoresACExists = true
					baseline.Power.MaxCoresACValue = uint32(val)
				}
				if val, _, err := xcKey.GetIntegerValue("DCSettingIndex"); err == nil {
					baseline.Power.MaxCoresDCExists = true
					baseline.Power.MaxCoresDCValue = uint32(val)
				}
				xcKey.Close()
			}
		}
		powerKey.Close()
	}

	// 5. BACK UP CRITICAL SERVICES (SysMain & wuauserv)
	targetServices := []string{"SysMain", "wuauserv"}
	for _, srv := range targetServices {
		srvPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\%s`, srv)
		if srvKey, err := registry.OpenKey(registry.LOCAL_MACHINE, srvPath, registry.QUERY_VALUE); err == nil {
			srvBackup := ServiceBackupState{ServiceName: srv}
			if val, _, err := srvKey.GetIntegerValue("Start"); err == nil {
				srvBackup.StartExists = true
				srvBackup.StartValue = uint32(val)
			}
			baseline.Services = append(baseline.Services, srvBackup)
			srvKey.Close()
		}
	}

	// 5b. BACK UP WIN32 PRIORITY SEPARATION (CPU QUANTUM)
	priorityControlPath := `SYSTEM\CurrentControlSet\Control\PriorityControl`
	if priKey, err := registry.OpenKey(registry.LOCAL_MACHINE, priorityControlPath, registry.QUERY_VALUE); err == nil {
		if val, _, err := priKey.GetIntegerValue("Win32PrioritySeparation"); err == nil {
			baseline.Win32PrioritySeparationExist = true
			baseline.Win32PrioritySeparationValue = uint32(val)
		}
		priKey.Close()
	}

	// 5c. BACK UP PERIPHERAL QUEUES
	mouseParamPath := `SYSTEM\CurrentControlSet\Services\mouclass\Parameters`
	if mKey, err := registry.OpenKey(registry.LOCAL_MACHINE, mouseParamPath, registry.QUERY_VALUE); err == nil {
		if val, _, err := mKey.GetIntegerValue("MouseDataQueueSize"); err == nil {
			baseline.MouseQueueExist = true
			baseline.MouseQueueValue = uint32(val)
		}
		mKey.Close()
	}

	keyboardParamPath := `SYSTEM\CurrentControlSet\Services\kbdclass\Parameters`
	if kKey, err := registry.OpenKey(registry.LOCAL_MACHINE, keyboardParamPath, registry.QUERY_VALUE); err == nil {
		if val, _, err := kKey.GetIntegerValue("KeyboardDataQueueSize"); err == nil {
			baseline.KeyboardQueueExist = true
			baseline.KeyboardQueueValue = uint32(val)
		}
		kKey.Close()
	}

	// 5d. BACK UP MOUSE SPEED AND ACCELERATION
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Mouse`, registry.QUERY_VALUE); err == nil {
		if val, _, err := key.GetStringValue("MouseSpeed"); err == nil {
			baseline.MouseSpeedExists = true
			baseline.MouseSpeedValue = val
		}
		if val, _, err := key.GetStringValue("MouseThreshold1"); err == nil {
			baseline.MouseThreshold1Exists = true
			baseline.MouseThreshold1Value = val
		}
		if val, _, err := key.GetStringValue("MouseThreshold2"); err == nil {
			baseline.MouseThreshold2Exists = true
			baseline.MouseThreshold2Value = val
		}
		key.Close()
	}

	// 5e. BACK UP KEYBOARD DELAY AND SPEED
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Keyboard`, registry.QUERY_VALUE); err == nil {
		if val, _, err := key.GetStringValue("KeyboardDelay"); err == nil {
			baseline.KeyboardDelayExists = true
			baseline.KeyboardDelayValue = val
		}
		if val, _, err := key.GetStringValue("KeyboardSpeed"); err == nil {
			baseline.KeyboardSpeedExists = true
			baseline.KeyboardSpeedValue = val
		}
		key.Close()
	}

	// 5f. BACK UP GAMEDVR CONFIG
	if key, err := registry.OpenKey(registry.CURRENT_USER, `System\GameConfigStore`, registry.QUERY_VALUE); err == nil {
		if val, _, err := key.GetIntegerValue("GameDVR_Enabled"); err == nil {
			baseline.GameDVREnabledExists = true
			baseline.GameDVREnabledValue = uint32(val)
		}
		key.Close()
	}
	if key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, registry.QUERY_VALUE); err == nil {
		if val, _, err := key.GetIntegerValue("AppCaptureEnabled"); err == nil {
			baseline.AppCaptureEnabledExists = true
			baseline.AppCaptureEnabledValue = uint32(val)
		}
		key.Close()
	}


	// 6. SERIALIZE AND WRITE BASELINE STATE TO FILE
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize baseline state: %w", err)
	}

	if err := os.WriteFile(BackupFileName, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write baseline state backup file: %w", err)
	}

	return baseline, nil
}

// LoadBaselineState reads the baseline backup file from disk.
func LoadBaselineState() (*SystemBaselineState, error) {
	data, err := os.ReadFile(BackupFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("baseline file does not exist: %w", err)
		}
		return nil, fmt.Errorf("failed to read baseline backup file: %w", err)
	}

	var baseline SystemBaselineState
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to deserialize backup file: %w", err)
	}

	return &baseline, nil
}

// RestoreBaselineState reads the active backup file and applies original settings back to the OS.
func RestoreBaselineState() error {
	if !isAdmin() {
		return errors.New("administrative privileges required to restore system baseline state")
	}

	baseline, err := LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline for rollback: %w", err)
	}

	// 1. RESTORE MSI DEVICE VALUES
	for _, dev := range baseline.Devices {
		fullPath := `SYSTEM\CurrentControlSet\Enum\PCI\` + dev.DevicePath
		
		// Restore MSISupported
		msiPath := fullPath + `\Device Parameters\Interrupt Management\MessageSignaledInterruptProperties`
		if msiKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, msiPath, registry.SET_VALUE); err == nil {
			if dev.MSISupportedExists {
				_ = msiKey.SetDWordValue("MSISupported", dev.MSISupportedValue)
			} else {
				_ = msiKey.DeleteValue("MSISupported")
			}
			msiKey.Close()
		}

		// Restore DevicePriority
		affPath := fullPath + `\Device Parameters\Interrupt Management\Affinity Policy`
		if affKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, affPath, registry.SET_VALUE); err == nil {
			if dev.DevicePriorityExists {
				_ = affKey.SetDWordValue("DevicePriority", dev.DevicePriorityValue)
			} else {
				_ = affKey.DeleteValue("DevicePriority")
			}
			affKey.Close()
		}
	}

	// 2. RESTORE SYSTEM LATENCY SETTINGS
	sysProfilePath := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	if sysKey, err := registry.OpenKey(registry.LOCAL_MACHINE, sysProfilePath, registry.SET_VALUE); err == nil {
		if baseline.Network.NetworkThrottlingExists {
			_ = sysKey.SetDWordValue("NetworkThrottlingIndex", baseline.Network.NetworkThrottlingValue)
		} else {
			_ = sysKey.DeleteValue("NetworkThrottlingIndex")
		}

		if baseline.Network.SystemResponsivenessExists {
			_ = sysKey.SetDWordValue("SystemResponsiveness", baseline.Network.SystemResponsivenessValue)
		} else {
			_ = sysKey.DeleteValue("SystemResponsiveness")
		}
		sysKey.Close()
	}

	// 3. RESTORE INTERFACE SETTINGS
	nicRootPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
	for _, nic := range baseline.Network.NICs {
		nicPath := nicRootPath + `\` + nic.InterfaceGUID
		if nicKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicPath, registry.SET_VALUE); err == nil {
			if nic.TcpAckFrequencyExists {
				_ = nicKey.SetDWordValue("TcpAckFrequency", nic.TcpAckFrequencyValue)
			} else {
				_ = nicKey.DeleteValue("TcpAckFrequency")
			}

			if nic.TCPNoDelayExists {
				_ = nicKey.SetDWordValue("TCPNoDelay", nic.TCPNoDelayValue)
			} else {
				_ = nicKey.DeleteValue("TCPNoDelay")
			}
			nicKey.Close()
		}
	}

	// 4. RESTORE ACTIVE POWER SCHEME GUID & CORE PARKING SETTINGS
	if baseline.Power.OriginalActiveScheme != "" {
		powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`
		if powerKey, err := registry.OpenKey(registry.LOCAL_MACHINE, powerPath, registry.SET_VALUE); err == nil {
			_ = powerKey.SetStringValue("ActivePowerScheme", baseline.Power.OriginalActiveScheme)
			powerKey.Close()
		}

		scheme := baseline.Power.OriginalActiveScheme

		// Restore Min Cores
		minCoresPath := fmt.Sprintf(`%s\%s\54533251-82be-4824-96c1-47b60b740d00\0cc5b647-c1df-4615-815a-8deb02312a2c`, powerPath, scheme)
		if mcKey, err := registry.OpenKey(registry.LOCAL_MACHINE, minCoresPath, registry.SET_VALUE); err == nil {
			if baseline.Power.MinCoresACExists {
				_ = mcKey.SetDWordValue("ACSettingIndex", baseline.Power.MinCoresACValue)
			} else {
				_ = mcKey.DeleteValue("ACSettingIndex")
			}
			if baseline.Power.MinCoresDCExists {
				_ = mcKey.SetDWordValue("DCSettingIndex", baseline.Power.MinCoresDCValue)
			} else {
				_ = mcKey.DeleteValue("DCSettingIndex")
			}
			mcKey.Close()
		}

		// Restore Max Cores
		maxCoresPath := fmt.Sprintf(`%s\%s\54533251-82be-4824-96c1-47b60b740d00\ea062031-0e34-4ff1-9b6d-eb1059334028`, powerPath, scheme)
		if xcKey, err := registry.OpenKey(registry.LOCAL_MACHINE, maxCoresPath, registry.SET_VALUE); err == nil {
			if baseline.Power.MaxCoresACExists {
				_ = xcKey.SetDWordValue("ACSettingIndex", baseline.Power.MaxCoresACValue)
			} else {
				_ = xcKey.DeleteValue("ACSettingIndex")
			}
			if baseline.Power.MaxCoresDCExists {
				_ = xcKey.SetDWordValue("DCSettingIndex", baseline.Power.MaxCoresDCValue)
			} else {
				_ = xcKey.DeleteValue("DCSettingIndex")
			}
			xcKey.Close()
		}

		// Reload power configuration to apply
		_ = system.Exec("powercfg", "/s", scheme)
	}

	// 5. RESTORE SERVICE STATES
	for _, srv := range baseline.Services {
		srvPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\%s`, srv.ServiceName)
		if srvKey, err := registry.OpenKey(registry.LOCAL_MACHINE, srvPath, registry.SET_VALUE); err == nil {
			if srv.StartExists {
				_ = srvKey.SetDWordValue("Start", srv.StartValue)
			} else {
				_ = srvKey.DeleteValue("Start")
			}
			srvKey.Close()
		}
	}

	// 5b. RESTORE WIN32 PRIORITY SEPARATION
	priorityControlPath := `SYSTEM\CurrentControlSet\Control\PriorityControl`
	if priKey, err := registry.OpenKey(registry.LOCAL_MACHINE, priorityControlPath, registry.SET_VALUE); err == nil {
		if baseline.Win32PrioritySeparationExist {
			_ = priKey.SetDWordValue("Win32PrioritySeparation", baseline.Win32PrioritySeparationValue)
		} else {
			_ = priKey.DeleteValue("Win32PrioritySeparation")
		}
		priKey.Close()
	}

	// 5c. RESTORE PERIPHERAL QUEUES
	mouseParamPath := `SYSTEM\CurrentControlSet\Services\mouclass\Parameters`
	if mKey, err := registry.OpenKey(registry.LOCAL_MACHINE, mouseParamPath, registry.SET_VALUE); err == nil {
		if baseline.MouseQueueExist {
			_ = mKey.SetDWordValue("MouseDataQueueSize", baseline.MouseQueueValue)
		} else {
			_ = mKey.DeleteValue("MouseDataQueueSize")
		}
		mKey.Close()
	}

	keyboardParamPath := `SYSTEM\CurrentControlSet\Services\kbdclass\Parameters`
	if kKey, err := registry.OpenKey(registry.LOCAL_MACHINE, keyboardParamPath, registry.SET_VALUE); err == nil {
		if baseline.KeyboardQueueExist {
			_ = kKey.SetDWordValue("KeyboardDataQueueSize", baseline.KeyboardQueueValue)
		} else {
			_ = kKey.DeleteValue("KeyboardDataQueueSize")
		}
		kKey.Close()
	}

	// 5d. RESTORE MOUSE SPEED AND ACCELERATION
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Mouse`, registry.SET_VALUE); err == nil {
		if baseline.MouseSpeedExists {
			_ = key.SetStringValue("MouseSpeed", baseline.MouseSpeedValue)
		}
		if baseline.MouseThreshold1Exists {
			_ = key.SetStringValue("MouseThreshold1", baseline.MouseThreshold1Value)
		}
		if baseline.MouseThreshold2Exists {
			_ = key.SetStringValue("MouseThreshold2", baseline.MouseThreshold2Value)
		}
		key.Close()
	}

	// 5e. RESTORE KEYBOARD DELAY AND SPEED
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Keyboard`, registry.SET_VALUE); err == nil {
		if baseline.KeyboardDelayExists {
			_ = key.SetStringValue("KeyboardDelay", baseline.KeyboardDelayValue)
		}
		if baseline.KeyboardSpeedExists {
			_ = key.SetStringValue("KeyboardSpeed", baseline.KeyboardSpeedValue)
		}
		key.Close()
	}

	// 5f. RESTORE GAMEDVR CONFIG
	if key, err := registry.OpenKey(registry.CURRENT_USER, `System\GameConfigStore`, registry.SET_VALUE); err == nil {
		if baseline.GameDVREnabledExists {
			_ = key.SetDWordValue("GameDVR_Enabled", baseline.GameDVREnabledValue)
		} else {
			_ = key.DeleteValue("GameDVR_Enabled")
		}
		key.Close()
	}
	if key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, registry.SET_VALUE); err == nil {
		if baseline.AppCaptureEnabledExists {
			_ = key.SetDWordValue("AppCaptureEnabled", baseline.AppCaptureEnabledValue)
		} else {
			_ = key.DeleteValue("AppCaptureEnabled")
		}
		key.Close()
	}

	return nil
}
