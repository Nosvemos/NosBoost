package hardware

import (
	"fmt"

	"nosboost/internal/system"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	DwmPerfKeyPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\dwm.exe\PerfOptions`
)

// SetDwmPriority elevates DWM priority to High (3) to eliminate borderless stutter, or reverts it (0 / delete).
func SetDwmPriority(enabled bool) error {
	if enabled {
		key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, DwmPerfKeyPath, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("failed to create dwm perf registry key: %w", err)
		}
		defer key.Close()

		if err := key.SetDWordValue("CpuPriorityClass", 3); err != nil {
			return fmt.Errorf("failed to set CpuPriorityClass: %w", err)
		}
	} else {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, DwmPerfKeyPath, registry.SET_VALUE)
		if err == nil {
			_ = key.DeleteValue("CpuPriorityClass")
			key.Close()
		}
	}
	return nil
}

// SetSearchSuspended disables/stops the Windows Search service to prevent background indexing stutters.
func SetSearchSuspended(suspended bool) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to SCM: %w", err)
	}
	defer m.Disconnect()

	srv, err := m.OpenService("WSearch")
	if err != nil {
		return fmt.Errorf("windows search service not found: %w", err)
	}
	defer srv.Close()

	if suspended {
		// Stop service first if running
		status, err := srv.Query()
		if err == nil && status.State != svc.Stopped {
			_, _ = srv.Control(svc.Stop)
		}
		// Set Startup to Disabled (4)
		_ = srv.UpdateConfig(mgr.Config{StartType: mgr.StartDisabled})
	} else {
		// Set Startup to Automatic Delayed (2)
		_ = srv.UpdateConfig(mgr.Config{StartType: mgr.StartAutomatic})
		// Start service
		_ = srv.Start()
	}

	return nil
}

// SetHibernationDisabled disables system hibernation via powercfg to save massive space and cut wake latency.
func SetHibernationDisabled(disabled bool) error {
	var stateStr string
	if disabled {
		stateStr = "off"
	} else {
		stateStr = "on"
	}

	if err := system.Exec("powercfg", "-h", stateStr); err != nil {
		return fmt.Errorf("failed to toggle hibernation state: %w", err)
	}
	return nil
}
