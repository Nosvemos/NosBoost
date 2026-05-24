package syswatch

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	// PROCESS_SUSPEND_RESUME is the Win32 handle access privilege for thread suspend commands.
	PROCESS_SUSPEND_RESUME = 0x0800
	SeDebugPrivilege       = "SeDebugPrivilege"
)

var (
	ntdll                  = windows.NewLazySystemDLL("ntdll.dll")
	procNtSuspendProcess  = ntdll.NewProc("NtSuspendProcess")
	procNtResumeProcess   = ntdll.NewProc("NtResumeProcess")

	// Mutex to protect active suspension handles lists
	suspensionMutex sync.Mutex
	suspendedPIDs   = make(map[string]uint32) // Key is service name, value is PID
)

// SetTokenPrivilege programmatically enables specific security privileges in our current token.
func SetTokenPrivilege(privilegeName string, enable bool) error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return fmt.Errorf("failed to open process token: %w", err)
	}
	defer token.Close()

	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(privilegeName), &luid)
	if err != nil {
		return fmt.Errorf("failed to lookup privilege luid %s: %w", privilegeName, err)
	}

	var tp windows.Tokenprivileges
	tp.PrivilegeCount = 1
	tp.Privileges[0].Luid = luid
	if enable {
		tp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	} else {
		tp.Privileges[0].Attributes = 0
	}

	err = windows.AdjustTokenPrivileges(token, false, &tp, 0, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to adjust token privileges: %w", err)
	}

	if err := windows.GetLastError(); err != nil {
		if errors.Is(err, windows.ERROR_NOT_ALL_ASSIGNED) {
			return fmt.Errorf("privilege %s not assigned to current token", privilegeName)
		}
	}

	return nil
}

// getServicePID connects to the SCM and queries the current active PID of a service.
func getServicePID(serviceName string) (uint32, error) {
	m, err := mgr.Connect()
	if err != nil {
		return 0, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return 0, err
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return 0, err
	}

	return status.ProcessId, nil
}

// ntSuspendProcess calls NtSuspendProcess from ntdll.dll safely.
func ntSuspendProcess(handle windows.Handle) error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from NtSuspendProcess panic: %v", r)
			}
		}()

		r1, _, e1 := procNtSuspendProcess.Call(uintptr(handle))
		if r1 != 0 {
			if e1 != nil && e1.Error() != "The operation completed successfully." {
				err = e1
			} else {
				err = fmt.Errorf("NTSTATUS error code: 0x%X", r1)
			}
		}
	}()
	return err
}

// ntResumeProcess calls NtResumeProcess from ntdll.dll safely.
func ntResumeProcess(handle windows.Handle) error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from NtResumeProcess panic: %v", r)
			}
		}()

		r1, _, e1 := procNtResumeProcess.Call(uintptr(handle))
		if r1 != 0 {
			if e1 != nil && e1.Error() != "The operation completed successfully." {
				err = e1
			} else {
				err = fmt.Errorf("NTSTATUS error code: 0x%X", r1)
			}
		}
	}()
	return err
}

// SuspendBackgroundServices enables debug privileges, queries PIDs for DiagTrack and wuauserv,
// and freezes their execution atomically.
func SuspendBackgroundServices() (int, error) {
	suspensionMutex.Lock()
	defer suspensionMutex.Unlock()

	// 1. Enable SeDebugPrivilege to elevate process handle query capabilities
	if err := SetTokenPrivilege(SeDebugPrivilege, true); err != nil {
		return 0, fmt.Errorf("failed to enable SeDebugPrivilege: %w", err)
	}

	targetServices := []string{"DiagTrack", "wuauserv"}
	suspendedCount := 0

	for _, srv := range targetServices {
		pid, err := getServicePID(srv)
		if err != nil || pid == 0 {
			continue // Skip if service is not active or stopped
		}

		// Open process handle with PROCESS_SUSPEND_RESUME access rights
		handle, err := windows.OpenProcess(PROCESS_SUSPEND_RESUME, false, pid)
		if err != nil {
			continue
		}

		// Freeze process thread queue
		if err := ntSuspendProcess(handle); err == nil {
			suspendedPIDs[srv] = pid
			suspendedCount++
		}
		windows.CloseHandle(handle)
	}

	return suspendedCount, nil
}

// ResumeBackgroundServices unfreezes all suspended services, returning them back to regular execution states.
func ResumeBackgroundServices() int {
	suspensionMutex.Lock()
	defer suspensionMutex.Unlock()

	// Enable SeDebugPrivilege
	_ = SetTokenPrivilege(SeDebugPrivilege, true)

	resumedCount := 0
	for srv, pid := range suspendedPIDs {
		handle, err := windows.OpenProcess(PROCESS_SUSPEND_RESUME, false, pid)
		if err == nil {
			if err := ntResumeProcess(handle); err == nil {
				resumedCount++
				delete(suspendedPIDs, srv)
			}
			windows.CloseHandle(handle)
		}
	}

	return resumedCount
}
