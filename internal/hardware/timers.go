package hardware

import (
	"fmt"
	"os/exec"
)

// OptimizeSystemTimers locks BCD parameters to enforce invariant TSC high-resolution timers.
func OptimizeSystemTimers() error {
	// 1. Disable platform clock (Forces reliance on low-overhead invariant TSC)
	cmd1 := exec.Command("bcdedit", "/set", "useplatformclock", "No")
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("failed to disable platform clock via bcdedit: %w", err)
	}

	// 2. Disable dynamic tick (Enforces constant CPU tick tracking, removing wake lag)
	cmd2 := exec.Command("bcdedit", "/set", "disabledynamictick", "Yes")
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("failed to disable dynamic tick via bcdedit: %w", err)
	}

	return nil
}

// RestoreSystemTimers deletes custom timer overrides, reverting Windows to standard scheduler defaults.
func RestoreSystemTimers() error {
	// 1. Remove useplatformclock override
	cmd1 := exec.Command("bcdedit", "/deletevalue", "useplatformclock")
	_ = cmd1.Run() // Ignore errors if the key was already absent

	// 2. Remove disabledynamictick override
	cmd2 := exec.Command("bcdedit", "/deletevalue", "disabledynamictick")
	_ = cmd2.Run()

	return nil
}
