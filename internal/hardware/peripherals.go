package hardware

import (
	"fmt"

	"nosboost/internal/config"

	"golang.org/x/sys/windows/registry"
)

const (
	MouseParametersKey    = `SYSTEM\CurrentControlSet\Services\mouclass\Parameters`
	KeyboardParametersKey = `SYSTEM\CurrentControlSet\Services\kbdclass\Parameters`
	TargetQueueSize       = 20 // Optimized queue buffer size (default is 100)
)

// TunePeripheralBuffers reduces class driver queue limits to enforce tighter, rapid packet flushes.
func TunePeripheralBuffers() error {
	// 1. Optimize Mouse queue buffer size
	mKey, err := registry.OpenKey(registry.LOCAL_MACHINE, MouseParametersKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open mouclass Parameters: %w", err)
	}
	defer mKey.Close()

	if err := mKey.SetDWordValue("MouseDataQueueSize", TargetQueueSize); err != nil {
		return fmt.Errorf("failed to set MouseDataQueueSize: %w", err)
	}

	// 2. Optimize Keyboard queue buffer size
	kKey, err := registry.OpenKey(registry.LOCAL_MACHINE, KeyboardParametersKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open kbdclass Parameters: %w", err)
	}
	defer kKey.Close()

	if err := kKey.SetDWordValue("KeyboardDataQueueSize", TargetQueueSize); err != nil {
		return fmt.Errorf("failed to set KeyboardDataQueueSize: %w", err)
	}

	return nil
}

// RestorePeripheralBuffers restores the original default Windows input queues from the baseline snapshot.
func RestorePeripheralBuffers() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	// 1. Restore Mouse Queue Size
	mKey, err := registry.OpenKey(registry.LOCAL_MACHINE, MouseParametersKey, registry.SET_VALUE)
	if err == nil {
		if baseline.MouseQueueExist {
			_ = mKey.SetDWordValue("MouseDataQueueSize", baseline.MouseQueueValue)
		} else {
			_ = mKey.DeleteValue("MouseDataQueueSize")
		}
		mKey.Close()
	}

	// 2. Restore Keyboard Queue Size
	kKey, err := registry.OpenKey(registry.LOCAL_MACHINE, KeyboardParametersKey, registry.SET_VALUE)
	if err == nil {
		if baseline.KeyboardQueueExist {
			_ = kKey.SetDWordValue("KeyboardDataQueueSize", baseline.KeyboardQueueValue)
		} else {
			_ = kKey.DeleteValue("KeyboardDataQueueSize")
		}
		kKey.Close()
	}

	return nil
}
