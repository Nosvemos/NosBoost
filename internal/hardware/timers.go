package hardware

import (
	"fmt"

	"nosboost/internal/system"
)

// OptimizeSystemTimers locks BCD parameters to enforce invariant TSC high-resolution timers.
func OptimizeSystemTimers() error {
	// 1. Disable platform clock (Forces reliance on low-overhead invariant TSC)
	if err := system.Exec("bcdedit", "/set", "useplatformclock", "No"); err != nil {
		return fmt.Errorf("failed to disable platform clock via bcdedit: %w", err)
	}

	// 2. Disable dynamic tick (Enforces constant CPU tick tracking, removing wake lag)
	if err := system.Exec("bcdedit", "/set", "disabledynamictick", "Yes"); err != nil {
		return fmt.Errorf("failed to disable dynamic tick via bcdedit: %w", err)
	}

	return nil
}

// RestoreSystemTimers deletes custom timer overrides, reverting Windows to standard scheduler defaults.
func RestoreSystemTimers() error {
	// 1. Remove useplatformclock override
	_ = system.Exec("bcdedit", "/deletevalue", "useplatformclock") // Ignore errors if the key was already absent

	// 2. Remove disabledynamictick override
	_ = system.Exec("bcdedit", "/deletevalue", "disabledynamictick")

	return nil
}
