package syswatch

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	GraphicsDriversKey = `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`
	GameBarKey         = `Software\Microsoft\GameBar`
)

// GetHAGSState checks the registry status of Hardware-Accelerated GPU Scheduling.
// Returns: 2 = Enabled, 1 = Disabled, 0 = Missing/Unsupported.
func GetHAGSState() (uint32, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, GraphicsDriversKey, registry.QUERY_VALUE)
	if err != nil {
		return 0, fmt.Errorf("failed to open GraphicsDrivers registry key: %w", err)
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("HwSchMode")
	if err != nil {
		return 0, nil // Key doesn't exist, which means HAGS is unsupported/disabled
	}

	return uint32(val), nil
}

// SetHAGSState updates HAGS registry state. Enabled sets HwSchMode = 2, Disabled sets to 1.
// Note: Activating or disabling HAGS requires a system reboot to apply changes.
func SetHAGSState(enabled bool) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, GraphicsDriversKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open GraphicsDrivers registry key: %w", err)
	}
	defer key.Close()

	var val uint32 = 1
	if enabled {
		val = 2
	}

	err = key.SetDWordValue("HwSchMode", val)
	if err != nil {
		return fmt.Errorf("failed to set HwSchMode registry parameter: %w", err)
	}

	return nil
}

// GetGameModeState checks if Windows Game Mode is enabled.
// Returns: true = Enabled, false = Disabled.
func GetGameModeState() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, GameBarKey, registry.QUERY_VALUE)
	if err != nil {
		return false, nil // Default is disabled or key not present
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("AllowAutoGameMode")
	if err != nil {
		return false, nil
	}

	return val == 1, nil
}

// SetGameModeState updates the registry parameter for Windows Auto Game Mode.
func SetGameModeState(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, GameBarKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open GameBar registry key: %w", err)
	}
	defer key.Close()

	var val uint32 = 0
	if enabled {
		val = 1
	}

	err = key.SetDWordValue("AllowAutoGameMode", val)
	if err != nil {
		return fmt.Errorf("failed to set AllowAutoGameMode registry parameter: %w", err)
	}

	return nil
}
