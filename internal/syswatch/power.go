package syswatch

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/sys/windows/registry"
)

const UltimateSchemeGUID = "e9a42b02-d5df-448d-aa00-03f14749eb61"

// getActiveScheme reads HKLM to find the currently active power scheme GUID.
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

// EnableUltimatePowerPlan duplicates the Ultimate Performance GUID if it is hidden in the system
// and locks it to active status.
func EnableUltimatePowerPlan() error {
	// 1. Incept/Duplicate Ultimate Performance power scheme if hidden
	dupCmd := exec.Command("powercfg", "-duplicatescheme", UltimateSchemeGUID)
	_ = dupCmd.Run() // May fail if it is already present, which is expected

	// 2. Force activate the Ultimate Performance scheme
	actCmd := exec.Command("powercfg", "-setactive", UltimateSchemeGUID)
	if err := actCmd.Run(); err != nil {
		return fmt.Errorf("failed to activate Ultimate Performance scheme: %w", err)
	}

	return nil
}

// StartPowerLockTicker spawns a background loop that monitors and enforces the Ultimate Performance
// power scheme every 5 seconds, preventing external scale-downs during gameplay.
func StartPowerLockTicker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				active, err := getActiveScheme()
				if err == nil && active != UltimateSchemeGUID {
					// Direct override and enforce Ultimate Performance plan
					_ = exec.Command("powercfg", "-setactive", UltimateSchemeGUID).Run()
				}
			}
		}
	}()
}
