package booster

import (
	"errors"
	"fmt"
	"os/exec"

	"nosboost/internal/config"

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
func EnableCoreParkingElimination() error {
	scheme, err := getActiveScheme()
	if err != nil {
		return fmt.Errorf("failed to detect active power plan: %w", err)
	}

	powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`
	
	// 1. Force Min Cores to 100
	minPath := fmt.Sprintf(`%s\%s\%s\%s`, powerPath, scheme, ProcessorSubgroup, MinCoresSetting)
	minKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, minPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open min cores power setting registry key: %w", err)
	}
	_ = minKey.SetDWordValue("ACSettingIndex", 100)
	_ = minKey.SetDWordValue("DCSettingIndex", 100)
	minKey.Close()

	// 2. Force Max Cores to 100
	maxPath := fmt.Sprintf(`%s\%s\%s\%s`, powerPath, scheme, ProcessorSubgroup, MaxCoresSetting)
	maxKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, maxPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open max cores power setting registry key: %w", err)
	}
	_ = maxKey.SetDWordValue("ACSettingIndex", 100)
	_ = maxKey.SetDWordValue("DCSettingIndex", 100)
	maxKey.Close()

	// 3. Trigger Windows to reload active power scheme index immediately
	cmd := exec.Command("powercfg", "/s", scheme)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload active power configuration scheme: %w", err)
	}

	return nil
}

// DisableCoreParkingElimination restores the original core parking settings of the active
// power scheme from our local baseline configuration database.
func DisableCoreParkingElimination() error {
	scheme, err := getActiveScheme()
	if err != nil {
		return fmt.Errorf("failed to detect active power plan: %w", err)
	}

	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	// Validate baseline power information is matching the target scheme
	if baseline.Power.OriginalActiveScheme != scheme {
		return errors.New("cannot restore core parking: active power plan differs from recorded baseline")
	}

	powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`

	// 1. Restore original Min Cores values
	minPath := fmt.Sprintf(`%s\%s\%s\%s`, powerPath, scheme, ProcessorSubgroup, MinCoresSetting)
	minKey, err := registry.OpenKey(registry.LOCAL_MACHINE, minPath, registry.SET_VALUE)
	if err == nil {
		if baseline.Power.MinCoresACExists {
			_ = minKey.SetDWordValue("ACSettingIndex", baseline.Power.MinCoresACValue)
		} else {
			_ = minKey.DeleteValue("ACSettingIndex")
		}

		if baseline.Power.MinCoresDCExists {
			_ = minKey.SetDWordValue("DCSettingIndex", baseline.Power.MinCoresDCValue)
		} else {
			_ = minKey.DeleteValue("DCSettingIndex")
		}
		minKey.Close()
	}

	// 2. Restore original Max Cores values
	maxPath := fmt.Sprintf(`%s\%s\%s\%s`, powerPath, scheme, ProcessorSubgroup, MaxCoresSetting)
	maxKey, err := registry.OpenKey(registry.LOCAL_MACHINE, maxPath, registry.SET_VALUE)
	if err == nil {
		if baseline.Power.MaxCoresACExists {
			_ = maxKey.SetDWordValue("ACSettingIndex", baseline.Power.MaxCoresACValue)
		} else {
			_ = maxKey.DeleteValue("ACSettingIndex")
		}

		if baseline.Power.MaxCoresDCExists {
			_ = maxKey.SetDWordValue("DCSettingIndex", baseline.Power.MaxCoresDCValue)
		} else {
			_ = maxKey.DeleteValue("DCSettingIndex")
		}
		maxKey.Close()
	}

	// 3. Trigger Windows to reload active power scheme index immediately
	cmd := exec.Command("powercfg", "/s", scheme)
	_ = cmd.Run()

	return nil
}
