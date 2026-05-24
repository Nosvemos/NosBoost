package memory

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// DisableSysMain connects to the service manager, disables SysMain (Superfetch)
// and forces the service instance to stop running immediately to minimize background page analysis.
func DisableSysMain() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("SysMain")
	if err != nil {
		return fmt.Errorf("failed to open SysMain service: %w", err)
	}
	defer s.Close()

	// 1. Set service startup type to Disabled
	config, err := s.Config()
	if err != nil {
		return fmt.Errorf("failed to query SysMain service configuration: %w", err)
	}

	if config.StartType != mgr.StartDisabled {
		err = s.UpdateConfig(mgr.Config{
			StartType: mgr.StartDisabled,
		})
		if err != nil {
			return fmt.Errorf("failed to disable SysMain startup type: %w", err)
		}
	}

	// 2. Stop running service instance if active
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query SysMain service status: %w", err)
	}

	if status.State != svc.Stopped && status.State != svc.StopPending {
		// Send Stop command
		_, err = s.Control(svc.Stop)
		if err != nil {
			return fmt.Errorf("failed to send stop command to SysMain service: %w", err)
		}

		// Wait for service to fully stop with a 5s timeout
		timeout := time.Now().Add(5 * time.Second)
		for {
			status, err = s.Query()
			if err != nil {
				break
			}
			if status.State == svc.Stopped {
				break
			}
			if time.Now().After(timeout) {
				return fmt.Errorf("timeout waiting for SysMain service to stop")
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	return nil
}

// RestoreSysMain restores the SysMain service back to its original startup configuration
// and restarts the service instance if it was running originally.
func RestoreSysMain(originalStartType uint32, wasRunning bool) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("SysMain")
	if err != nil {
		return fmt.Errorf("failed to open SysMain service: %w", err)
	}
	defer s.Close()

	// 1. Restore startup type
	err = s.UpdateConfig(mgr.Config{
		StartType: originalStartType,
	})
	if err != nil {
		return fmt.Errorf("failed to restore SysMain start configuration: %w", err)
	}

	// 2. Restart the service if it was originally running
	if wasRunning {
		status, err := s.Query()
		if err == nil && status.State != svc.Running && status.State != svc.StartPending {
			err = s.Start()
			if err != nil {
				return fmt.Errorf("failed to restart SysMain service: %w", err)
			}
		}
	}

	return nil
}
