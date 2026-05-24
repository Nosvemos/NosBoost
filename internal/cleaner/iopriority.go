package cleaner

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// ProcessIoPriority is the native NT information class index.
	ProcessIoPriority = 33

	// IO_PRIORITY_HINT enums
	IoPriorityVeryLow = 0
	IoPriorityLow     = 1
	IoPriorityNormal  = 2
)

var (
	ntdll                      = windows.NewLazySystemDLL("ntdll.dll")
	procNtSetInformationProcess = ntdll.NewProc("NtSetInformationProcess")

	// Mutex to protect active throttled process records
	ioMutex      sync.Mutex
	throttledPIDs = make(map[uint32]bool)

	// Noisy background applications we target for I/O throttling
	IOBackgroundTargets = []string{
		"discord.exe",
		"spotify.exe",
		"chrome.exe",
		"msedge.exe",
		"firefox.exe",
		"steamwebhelper.exe",
		"epicgameslauncher.exe",
		"battlenet.exe",
		"galaxyclient.exe",
		"steam.exe",
		"origin.exe",
	}
)

// ntSetInformationProcess invokes NtSetInformationProcess from ntdll.dll safely.
func ntSetInformationProcess(handle windows.Handle, class uint32, info unsafe.Pointer, length uint32) error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from NtSetInformationProcess panic: %v", r)
			}
		}()

		r1, _, e1 := procNtSetInformationProcess.Call(
			uintptr(handle),
			uintptr(class),
			uintptr(info),
			uintptr(length),
		)
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

// ThrottleBackgroundIO discovers active background processes and scales their disk I/O priority down to Low (1).
func ThrottleBackgroundIO() (int, error) {
	ioMutex.Lock()
	defer ioMutex.Unlock()

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to capture process snapshot for IO prioritization: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0, fmt.Errorf("failed to read first process entry: %w", err)
	}

	targetMap := make(map[string]bool)
	for _, t := range IOBackgroundTargets {
		targetMap[strings.ToLower(t)] = true
	}

	throttledCount := 0
	priority := uint32(IoPriorityLow) // Set to Low I/O Priority (Background)

	for {
		exeName := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if targetMap[exeName] {
			pid := entry.ProcessID
			// Open process with PROCESS_SET_INFORMATION access rights
			handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, pid)
			if err == nil {
				// Invoke low-level NT ProcessIoPriority class
				err = ntSetInformationProcess(handle, ProcessIoPriority, unsafe.Pointer(&priority), uint32(unsafe.Sizeof(priority)))
				if err == nil {
					throttledPIDs[pid] = true
					throttledCount++
				}
				windows.CloseHandle(handle)
			}
		}

		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			break
		}
	}

	return throttledCount, nil
}

// RestoreBackgroundIO restores all throttled background processes back to Normal (2) I/O priority.
func RestoreBackgroundIO() int {
	ioMutex.Lock()
	defer ioMutex.Unlock()

	restoredCount := 0
	priority := uint32(IoPriorityNormal) // Restore to Normal I/O Priority

	for pid := range throttledPIDs {
		handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, pid)
		if err == nil {
			err = ntSetInformationProcess(handle, ProcessIoPriority, unsafe.Pointer(&priority), uint32(unsafe.Sizeof(priority)))
			if err == nil {
				restoredCount++
				delete(throttledPIDs, pid)
			}
			windows.CloseHandle(handle)
		}
	}

	return restoredCount
}
