package booster

import (
	"errors"
	"fmt"
	"strconv"

	"nosboost/internal/config"
	"nosboost/internal/system"

	"golang.org/x/sys/windows/registry"
)

const (
	ProcessorSubgroup = "54533251-82be-4824-96c1-47b60b740d00"
	MinCoresSetting   = "0cc5b647-c1df-4615-815a-8deb02312a2c"
	MaxCoresSetting   = "ea062031-0e34-4ff1-9b6d-eb1059334028"
)

// getActiveScheme retrieves the GUID of the currently active Windows power plan.
func getActiveScheme() (string, error) {
	powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`
	powerKey, err := registry.OpenKey(registry.LOCAL_MACHINE, powerPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open power schemes key: %w", err)
	}
	defer powerKey.Close()

	scheme, _, err := powerKey.GetStringValue("ActivePowerScheme")
	if err != nil {
		return "", fmt.Errorf("failed to read ActivePowerScheme: %w", err)
	}
	return scheme, nil
}

// EnableCoreParkingElimination forces the minimum and maximum CPU core limits to 100%
// for the currently active power scheme, permanently keeping all physical/logical cores awake.
// Uses powercfg command-line calls to bypass strict SYSTEM registry write blocks.
func EnableCoreParkingElimination() error {
	scheme, err := getActiveScheme()
	if err != nil {
		return fmt.Errorf("failed to detect active power plan: %w", err)
	}

	// 1. Force Min Cores to 100 (AC and DC)
	_ = system.Exec("powercfg", "/setacvalueindex", scheme, ProcessorSubgroup, MinCoresSetting, "100")
	_ = system.Exec("powercfg", "/setdcvalueindex", scheme, ProcessorSubgroup, MinCoresSetting, "100")

	// 2. Force Max Cores to 100 (AC and DC)
	_ = system.Exec("powercfg", "/setacvalueindex", scheme, ProcessorSubgroup, MaxCoresSetting, "100")
	_ = system.Exec("powercfg", "/setdcvalueindex", scheme, ProcessorSubgroup, MaxCoresSetting, "100")

	// 3. Trigger Windows to reload active power scheme index immediately to apply
	if err := system.Exec("powercfg", "/setactive", scheme); err != nil {
		return fmt.Errorf("failed to reload active power configuration scheme: %w", err)
	}

	return nil
}

// DisableCoreParkingElimination restores the original core parking settings of the active
// power scheme from our local baseline configuration database using powercfg utility.
func DisableCoreParkingElimination() error {
	scheme, err := getActiveScheme()
	if err != nil {
		return fmt.Errorf("failed to detect active power plan: %w", err)
	}

	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	// Validate baseline power information matches the target scheme
	if baseline.Power.OriginalActiveScheme != scheme {
		return errors.New("cannot restore core parking: active power plan differs from recorded baseline")
	}

	// 1. Restore original Min Cores values
	if baseline.Power.MinCoresACExists {
		_ = system.Exec("powercfg", "/setacvalueindex", scheme, ProcessorSubgroup, MinCoresSetting, strconv.FormatUint(uint64(baseline.Power.MinCoresACValue), 10))
	}
	if baseline.Power.MinCoresDCExists {
		_ = system.Exec("powercfg", "/setdcvalueindex", scheme, ProcessorSubgroup, MinCoresSetting, strconv.FormatUint(uint64(baseline.Power.MinCoresDCValue), 10))
	}

	// 2. Restore original Max Cores values
	if baseline.Power.MaxCoresACExists {
		_ = system.Exec("powercfg", "/setacvalueindex", scheme, ProcessorSubgroup, MaxCoresSetting, strconv.FormatUint(uint64(baseline.Power.MaxCoresACValue), 10))
	}
	if baseline.Power.MaxCoresDCExists {
		_ = system.Exec("powercfg", "/setdcvalueindex", scheme, ProcessorSubgroup, MaxCoresSetting, strconv.FormatUint(uint64(baseline.Power.MaxCoresDCValue), 10))
	}

	// 3. Trigger Windows to reload active power scheme index immediately
	_ = system.Exec("powercfg", "/setactive", scheme)

	return nil
}
