package hardware

import (
	"fmt"

	"nosboost/internal/config"
	"nosboost/internal/system"

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

// getActiveScheme retrieves the GUID of the currently active Windows power plan.
func getActiveScheme() (string, error) {
	powerPath := `SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes`
	powerKey, err := registry.OpenKey(registry.LOCAL_MACHINE, powerPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer powerKey.Close()
	scheme, _, err := powerKey.GetStringValue("ActivePowerScheme")
	return scheme, err
}

// OptimizeInputLatency applies deep input latency reductions (mouse, keyboard delay, Game DVR, USB selective suspend).
func OptimizeInputLatency() error {
	// 1. Disable Windows Mouse Acceleration (1:1 Raw Input)
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Mouse`, registry.SET_VALUE); err == nil {
		_ = key.SetStringValue("MouseSpeed", "0")
		_ = key.SetStringValue("MouseThreshold1", "0")
		_ = key.SetStringValue("MouseThreshold2", "0")
		key.Close()
	}

	// 2. Maximize Keyboard Repeat Speed and Minimize Delay
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Keyboard`, registry.SET_VALUE); err == nil {
		_ = key.SetStringValue("KeyboardDelay", "0")
		_ = key.SetStringValue("KeyboardSpeed", "31")
		key.Close()
	}

	// 3. Disable Game DVR & App Capture (eliminate background overlays and stutters)
	if key, _, err := registry.CreateKey(registry.CURRENT_USER, `System\GameConfigStore`, registry.SET_VALUE); err == nil {
		_ = key.SetDWordValue("GameDVR_Enabled", 0)
		key.Close()
	}
	if key, _, err := registry.CreateKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, registry.SET_VALUE); err == nil {
		_ = key.SetDWordValue("AppCaptureEnabled", 0)
		key.Close()
	}
	if key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\GameDVR`, registry.SET_VALUE); err == nil {
		_ = key.SetDWordValue("AllowGameDVR", 0)
		key.Close()
	}

	// 4. Disable USB Selective Suspend in current power scheme (prevents device stutters)
	if scheme, err := getActiveScheme(); err == nil {
		_ = system.Exec("powercfg", "/setacvalueindex", scheme, "2a737441-1930-4402-8d77-b2bebba308a3", "48e6d7a4-450f-48d2-957c-d4924c350322", "0")
		_ = system.Exec("powercfg", "/setdcvalueindex", scheme, "2a737441-1930-4402-8d77-b2bebba308a3", "48e6d7a4-450f-48d2-957c-d4924c350322", "0")
		_ = system.Exec("powercfg", "/setactive", scheme)
	}

	return nil
}

// RestoreInputLatency rolls back Mouse acceleration, Keyboard delays, Game DVR, and USB selective suspend to OS defaults.
func RestoreInputLatency() error {
	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	// 1. Restore Mouse Acceleration Speed parameters
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Mouse`, registry.SET_VALUE); err == nil {
		if baseline.MouseSpeedExists {
			_ = key.SetStringValue("MouseSpeed", baseline.MouseSpeedValue)
		} else {
			_ = key.DeleteValue("MouseSpeed")
		}
		if baseline.MouseThreshold1Exists {
			_ = key.SetStringValue("MouseThreshold1", baseline.MouseThreshold1Value)
		} else {
			_ = key.DeleteValue("MouseThreshold1")
		}
		if baseline.MouseThreshold2Exists {
			_ = key.SetStringValue("MouseThreshold2", baseline.MouseThreshold2Value)
		} else {
			_ = key.DeleteValue("MouseThreshold2")
		}
		key.Close()
	}

	// 2. Restore Keyboard Repeat Settings
	if key, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Keyboard`, registry.SET_VALUE); err == nil {
		if baseline.KeyboardDelayExists {
			_ = key.SetStringValue("KeyboardDelay", baseline.KeyboardDelayValue)
		} else {
			_ = key.DeleteValue("KeyboardDelay")
		}
		if baseline.KeyboardSpeedExists {
			_ = key.SetStringValue("KeyboardSpeed", baseline.KeyboardSpeedValue)
		} else {
			_ = key.DeleteValue("KeyboardSpeed")
		}
		key.Close()
	}

	// 3. Restore Game DVR & App Capture Settings
	if key, err := registry.OpenKey(registry.CURRENT_USER, `System\GameConfigStore`, registry.SET_VALUE); err == nil {
		if baseline.GameDVREnabledExists {
			_ = key.SetDWordValue("GameDVR_Enabled", baseline.GameDVREnabledValue)
		} else {
			_ = key.DeleteValue("GameDVR_Enabled")
		}
		key.Close()
	}
	if key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, registry.SET_VALUE); err == nil {
		if baseline.AppCaptureEnabledExists {
			_ = key.SetDWordValue("AppCaptureEnabled", baseline.AppCaptureEnabledValue)
		} else {
			_ = key.DeleteValue("AppCaptureEnabled")
		}
		key.Close()
	}
	if key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\GameDVR`, registry.SET_VALUE); err == nil {
		_ = key.DeleteValue("AllowGameDVR")
		key.Close()
	}

	// 4. Restore USB Selective Suspend (normally 1 / Enabled on standard OS)
	if scheme, err := getActiveScheme(); err == nil {
		_ = system.Exec("powercfg", "/setacvalueindex", scheme, "2a737441-1930-4402-8d77-b2bebba308a3", "48e6d7a4-450f-48d2-957c-d4924c350322", "1")
		_ = system.Exec("powercfg", "/setdcvalueindex", scheme, "2a737441-1930-4402-8d77-b2bebba308a3", "48e6d7a4-450f-48d2-957c-d4924c350322", "1")
		_ = system.Exec("powercfg", "/setactive", scheme)
	}

	return nil
}

