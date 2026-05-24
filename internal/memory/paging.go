package memory

import (
	"fmt"

	"nosboost/internal/config"

	"golang.org/x/sys/windows/registry"
)

const (
	MemoryManagementKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`
)

// OptimizePagingExecutive disables paging out the kernel executive and system drivers to RAM,
// and sets large system cache to disabled to yield maximum free cache space to active games.
// This completely resolves driver paging stutters (micro-stutters) and stabilizes 99% / 95% frame lows.
func OptimizePagingExecutive() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, MemoryManagementKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open memory management key: %w", err)
	}
	defer key.Close()

	// Disable paging executive (forces kernel & drivers to stay in physical RAM)
	if err := key.SetDWordValue("DisablePagingExecutive", 1); err != nil {
		return fmt.Errorf("failed to disable paging executive: %w", err)
	}

	// Keep large system cache compact (leaves max free space for games)
	if err := key.SetDWordValue("LargeSystemCache", 0); err != nil {
		return fmt.Errorf("failed to set large system cache: %w", err)
	}

	return nil
}

// RestorePagingExecutive reverts the paging executive and system cache settings back to baseline.
func RestorePagingExecutive() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, MemoryManagementKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open memory management key: %w", err)
	}
	defer key.Close()

	if baseline.DisablePagingExecutiveExists {
		_ = key.SetDWordValue("DisablePagingExecutive", baseline.DisablePagingExecutiveValue)
	} else {
		_ = key.DeleteValue("DisablePagingExecutive")
	}

	if baseline.LargeSystemCacheExists {
		_ = key.SetDWordValue("LargeSystemCache", baseline.LargeSystemCacheValue)
	} else {
		_ = key.DeleteValue("LargeSystemCache")
	}

	return nil
}
