package memory

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	psapi               = windows.NewLazySystemDLL("psapi.dll")
	procEmptyWorkingSet = psapi.NewProc("EmptyWorkingSet")

	// Critical system processes that should never be trimmed to avoid OS instability.
	SystemCriticalProcesses = map[string]bool{
		"system":             true,
		"idle":               true,
		"smss.exe":           true,
		"csrss.exe":          true,
		"wininit.exe":        true,
		"services.exe":       true,
		"lsass.exe":          true,
		"svchost.exe":        true,
		"winlogon.exe":       true,
		"spoolsv.exe":        true,
		"dwm.exe":            true,
		"explorer.exe":       true,
		"securityhealthservice.exe": true,
	}
)

// emptyWorkingSet wraps the PSAPI EmptyWorkingSet call.
func emptyWorkingSet(handle windows.Handle) error {
	r1, _, err := procEmptyWorkingSet.Call(uintptr(handle))
	if r1 == 0 {
		return err
	}
	return nil
}

// TrimWorkingSets flushes working sets of background applications to reclaim physical RAM.
// It bypasses NosBoost itself, the active game PID, and critical system binaries.
func TrimWorkingSets(gamePID uint32) (int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to capture process snapshot for trimming: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0, fmt.Errorf("failed to read first process entry: %w", err)
	}

	myPID := uint32(os.Getpid())
	trimmedCount := 0

	for {
		pid := entry.ProcessID
		exeName := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))

		// Exclude guards
		isExcluded := pid == 0 || pid == myPID || pid == gamePID || SystemCriticalProcesses[exeName]

		if !isExcluded {
			// Open process with PROCESS_SET_QUOTA access rights
			handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
			if err == nil {
				// Flush memory pages to pagefile
				if err := emptyWorkingSet(handle); err == nil {
					trimmedCount++
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

	return trimmedCount, nil
}
