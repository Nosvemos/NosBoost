package booster

import (
	"fmt"

	"nosboost/internal/config"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	PriorityControlKey             = `SYSTEM\CurrentControlSet\Control\PriorityControl`
	Win32PrioritySeparationOptimal = 0x26 // Short, variable quantums with high foreground ratio
)

// ElevateProcessPriority changes the scheduling priority of the game to High Priority Class.
// It returns the original priority class to enable perfect state rollback.
func ElevateProcessPriority(pid uint32) (uint32, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_SET_INFORMATION, false, pid)
	if err != nil {
		return 0, fmt.Errorf("failed to open game process for priority control: %w", err)
	}
	defer windows.CloseHandle(handle)

	// Read current priority class
	originalPriority, err := windows.GetPriorityClass(handle)
	if err != nil {
		return 0, fmt.Errorf("failed to query game process priority class: %w", err)
	}

	// Elevate priority class to HIGH_PRIORITY_CLASS (0x00000080)
	err = windows.SetPriorityClass(handle, windows.HIGH_PRIORITY_CLASS)
	if err != nil {
		return 0, fmt.Errorf("failed to set game process priority class to High: %w", err)
	}

	return originalPriority, nil
}

// RestoreProcessPriority rolls back the target game's scheduling priority class.
func RestoreProcessPriority(pid uint32, originalPriority uint32) error {
	if originalPriority == 0 {
		return nil
	}

	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, pid)
	if err != nil {
		return fmt.Errorf("failed to open process for priority restoration: %w", err)
	}
	defer windows.CloseHandle(handle)

	err = windows.SetPriorityClass(handle, originalPriority)
	if err != nil {
		return fmt.Errorf("failed to restore process priority class: %w", err)
	}

	return nil
}

// OptimizePrioritySeparation sets Win32PrioritySeparation to the optimized competitive gaming value (0x26).
func OptimizePrioritySeparation() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, PriorityControlKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open PriorityControl key: %w", err)
	}
	defer key.Close()

	err = key.SetDWordValue("Win32PrioritySeparation", Win32PrioritySeparationOptimal)
	if err != nil {
		return fmt.Errorf("failed to set Win32PrioritySeparation: %w", err)
	}

	return nil
}

// RestorePrioritySeparation restores Win32PrioritySeparation to the baseline value recorded during system snapshot.
func RestorePrioritySeparation() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, PriorityControlKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open PriorityControl key: %w", err)
	}
	defer key.Close()

	if baseline.Win32PrioritySeparationExist {
		err = key.SetDWordValue("Win32PrioritySeparation", baseline.Win32PrioritySeparationValue)
	} else {
		err = key.DeleteValue("Win32PrioritySeparation")
	}

	if err != nil {
		return fmt.Errorf("failed to restore Win32PrioritySeparation: %w", err)
	}

	return nil
}

// OptimizeGamesTask configures the MMCSS Games task priority values to maximize CPU and GPU thread scheduling for games.
// This significantly stabilizes 99% and 95% frame lows by prioritizing game execution over background services.
func OptimizeGamesTask() error {
	gamesTaskPath := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games`
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, gamesTaskPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open/create MMCSS Games registry key: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue("Affinity", 0); err != nil {
		return fmt.Errorf("failed to set MMCSS Affinity: %w", err)
	}
	if err := key.SetStringValue("Background Only", "False"); err != nil {
		return fmt.Errorf("failed to set MMCSS Background Only: %w", err)
	}
	if err := key.SetDWordValue("Clock Rate", 10000); err != nil {
		return fmt.Errorf("failed to set MMCSS Clock Rate: %w", err)
	}
	if err := key.SetDWordValue("GPU Priority", 8); err != nil {
		return fmt.Errorf("failed to set MMCSS GPU Priority: %w", err)
	}
	if err := key.SetDWordValue("Priority", 6); err != nil {
		return fmt.Errorf("failed to set MMCSS Priority: %w", err)
	}
	if err := key.SetStringValue("Scheduling Category", "High"); err != nil {
		return fmt.Errorf("failed to set MMCSS Scheduling Category: %w", err)
	}
	if err := key.SetStringValue("SFIO Priority", "High"); err != nil {
		return fmt.Errorf("failed to set MMCSS SFIO Priority: %w", err)
	}

	return nil
}

// RestoreGamesTask restores the MMCSS Games task priority values from baseline.
func RestoreGamesTask() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	gamesTaskPath := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, gamesTaskPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open MMCSS Games key: %w", err)
	}
	defer key.Close()

	if baseline.GamesTask.AffinityExists {
		_ = key.SetDWordValue("Affinity", baseline.GamesTask.AffinityValue)
	} else {
		_ = key.DeleteValue("Affinity")
	}

	if baseline.GamesTask.BackgroundOnlyExists {
		_ = key.SetStringValue("Background Only", baseline.GamesTask.BackgroundOnlyValue)
	} else {
		_ = key.DeleteValue("Background Only")
	}

	if baseline.GamesTask.ClockRateExists {
		_ = key.SetDWordValue("Clock Rate", baseline.GamesTask.ClockRateValue)
	} else {
		_ = key.DeleteValue("Clock Rate")
	}

	if baseline.GamesTask.GPUPriorityExists {
		_ = key.SetDWordValue("GPU Priority", baseline.GamesTask.GPUPriorityValue)
	} else {
		_ = key.DeleteValue("GPU Priority")
	}

	if baseline.GamesTask.PriorityExists {
		_ = key.SetDWordValue("Priority", baseline.GamesTask.PriorityValue)
	} else {
		_ = key.DeleteValue("Priority")
	}

	if baseline.GamesTask.SchedulingCategoryExists {
		_ = key.SetStringValue("Scheduling Category", baseline.GamesTask.SchedulingCategoryValue)
	} else {
		_ = key.DeleteValue("Scheduling Category")
	}

	if baseline.GamesTask.SFIOPriorityExists {
		_ = key.SetStringValue("SFIO Priority", baseline.GamesTask.SFIOPriorityValue)
	} else {
		_ = key.DeleteValue("SFIO Priority")
	}

	return nil
}
