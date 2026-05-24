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

var ActiveTargetScheme = UltimateSchemeGUID

// EnableUltimatePowerPlan duplicates the Ultimate Performance GUID if it is hidden in the system
// and locks it to active status. If Ultimate is not supported, it cascades to High Performance.
func EnableUltimatePowerPlan() error {
	// 1. Incept/Duplicate Ultimate Performance power scheme if hidden
	_ = exec.Command("powercfg", "-duplicatescheme", UltimateSchemeGUID).Run()

	// 2. Force activate the Ultimate Performance scheme
	if err := exec.Command("powercfg", "-setactive", UltimateSchemeGUID).Run(); err == nil {
		ActiveTargetScheme = UltimateSchemeGUID
		return nil
	}

	// 3. Fallback to High Performance scheme (Standard)
	highPerformanceGUID := "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
	if err := exec.Command("powercfg", "-setactive", highPerformanceGUID).Run(); err == nil {
		ActiveTargetScheme = highPerformanceGUID
		return nil
	}

	// 4. Ultimate fail-safe fallback: keep the current active plan locked
	if active, err := getActiveScheme(); err == nil {
		ActiveTargetScheme = active
		return nil
	}

	return fmt.Errorf("failed to activate any high-performance power schemes")
}

// StartPowerLockTicker spawns a background loop that monitors and enforces our active target
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
				if err == nil && active != ActiveTargetScheme {
					// Direct override and enforce targeted performance plan
					_ = exec.Command("powercfg", "-setactive", ActiveTargetScheme).Run()
				}
			}
		}
	}()
}
