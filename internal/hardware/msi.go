package hardware

import (
	"fmt"

	"nosboost/internal/config"

	"golang.org/x/sys/windows/registry"
)

// scanActivePCIDevices iterates over the PCI Enum registry tree and filters active displaying GPUs and network NICs.
func scanActivePCIDevices() ([]string, error) {
	pciKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Enum\PCI`, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open PCI registry tree: %w", err)
	}
	defer pciKey.Close()

	vendors, err := pciKey.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read PCI vendor keys: %w", err)
	}

	var activeDevices []string
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
			devPath := vendor + `\` + instance
			fullPath := `SYSTEM\CurrentControlSet\Enum\PCI\` + devPath

			// Check our Three-Factor Active Validation Model
			key, err := registry.OpenKey(registry.LOCAL_MACHINE, fullPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			// 1. ClassGUID match Display or Net adapter
			classGUID, _, err := key.GetStringValue("ClassGUID")
			if err != nil {
				key.Close()
				continue
			}

			isTargetClass := classGUID == "{4d36e968-e325-11ce-bfc1-08002be10318}" || classGUID == "{4d36e972-e325-11ce-bfc1-08002be10318}"
			if !isTargetClass {
				key.Close()
				continue
			}

			// 2. Active service verification (Service matches a loaded driver)
			service, _, err := key.GetStringValue("Service")
			if err != nil || service == "" {
				key.Close()
				continue
			}

			// 3. ConfigFlags validation (ConfigFlags == 0 indicates active, online, started hardware)
			configFlags, _, err := key.GetIntegerValue("ConfigFlags")
			if err != nil || configFlags != 0 {
				key.Close()
				continue
			}

			key.Close()
			activeDevices = append(activeDevices, devPath)
		}
	}
	return activeDevices, nil
}

// EnableMSIMode traversing PCI devices, validating active ones, and converting them to MSI mode with High Priority.
func EnableMSIMode() error {
	devices, err := scanActivePCIDevices()
	if err != nil {
		return err
	}

	for _, devPath := range devices {
		basePath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Enum\PCI\%s\Device Parameters\Interrupt Management`, devPath)

		// 1. Enforce MSISupported = 1
		msiPath := basePath + `\MessageSignaledInterruptProperties`
		msiKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, msiPath, registry.SET_VALUE)
		if err != nil {
			continue
		}
		_ = msiKey.SetDWordValue("MSISupported", 1)
		msiKey.Close()

		// 2. Enforce DevicePriority = 3 (High Priority)
		affPath := basePath + `\Affinity Policy`
		affKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, affPath, registry.SET_VALUE)
		if err != nil {
			continue
		}
		_ = affKey.SetDWordValue("DevicePriority", 3)
		affKey.Close()
	}

	return nil
}

// DisableMSIMode restores the original MSI mode settings of active Display and Network devices from our baseline snapshot.
func DisableMSIMode() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline configuration: %w", err)
	}

	for _, dev := range baseline.Devices {
		basePath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Enum\PCI\%s\Device Parameters\Interrupt Management`, dev.DevicePath)

		// Restore MSISupported
		msiPath := basePath + `\MessageSignaledInterruptProperties`
		if msiKey, err := registry.OpenKey(registry.LOCAL_MACHINE, msiPath, registry.SET_VALUE); err == nil {
			if dev.MSISupportedExists {
				_ = msiKey.SetDWordValue("MSISupported", dev.MSISupportedValue)
			} else {
				_ = msiKey.DeleteValue("MSISupported")
			}
			msiKey.Close()
		}

		// Restore DevicePriority
		affPath := basePath + `\Affinity Policy`
		if affKey, err := registry.OpenKey(registry.LOCAL_MACHINE, affPath, registry.SET_VALUE); err == nil {
			if dev.DevicePriorityExists {
				_ = affKey.SetDWordValue("DevicePriority", dev.DevicePriorityValue)
			} else {
				_ = affKey.DeleteValue("DevicePriority")
			}
			affKey.Close()
		}
	}

	return nil
}
