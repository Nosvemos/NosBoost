package booster

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Map of PID to its original affinity mask, to support clean restoration.
	originalBackgroundAffinities = make(map[uint32]uintptr)
	affinityMutex                sync.Mutex

	// Common background processes we target for isolation to Core 0 & 1.
	BackgroundTargets = []string{
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
		"lghub.exe",
		"razercentral.exe",
	}

	// Dynamic DLL loads for low-level kernel32 processes
	kernel32                    = windows.NewLazySystemDLL("kernel32.dll")
	procGetProcessAffinityMask = kernel32.NewProc("GetProcessAffinityMask")
	procSetProcessAffinityMask = kernel32.NewProc("SetProcessAffinityMask")
)

// getProcessAffinityMask wraps GetProcessAffinityMask from kernel32.dll.
func getProcessAffinityMask(handle windows.Handle) (uintptr, uintptr, error) {
	var procAff, sysAff uintptr
	r1, _, err := procGetProcessAffinityMask.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&procAff)),
		uintptr(unsafe.Pointer(&sysAff)),
	)
	if r1 == 0 {
		return 0, 0, err
	}
	return procAff, sysAff, nil
}

// setProcessAffinityMask wraps SetProcessAffinityMask from kernel32.dll.
func setProcessAffinityMask(handle windows.Handle, mask uintptr) error {
	r1, _, err := procSetProcessAffinityMask.Call(
		uintptr(handle),
		mask,
	)
	if r1 == 0 {
		return err
	}
	return nil
}

// getProcessAffinity queries the active process and system affinity bitmasks.
func getProcessAffinity(pid uint32) (uintptr, uintptr, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return 0, 0, err
	}
	defer windows.CloseHandle(handle)

	return getProcessAffinityMask(handle)
}

// setProcessAffinity applies a specific core affinity bitmask to the target process.
func setProcessAffinity(pid uint32, mask uintptr) error {
	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return setProcessAffinityMask(handle, mask)
}

// RestrictBackgroundProcesses binds all discovered noisy background processes to Core 0 and 1.
func RestrictBackgroundProcesses() error {
	affinityMutex.Lock()
	defer affinityMutex.Unlock()

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return fmt.Errorf("failed to capture process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return fmt.Errorf("failed to read first process entry: %w", err)
	}

	targetMap := make(map[string]bool)
	for _, t := range BackgroundTargets {
		targetMap[strings.ToLower(t)] = true
	}

	for {
		exeName := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if targetMap[exeName] {
			pid := entry.ProcessID
			procAff, sysAff, err := getProcessAffinity(pid)
			if err == nil {
				// Record original state if not already logged
				if _, exists := originalBackgroundAffinities[pid]; !exists {
					originalBackgroundAffinities[pid] = procAff
				}
				// Bind only to Core 0 and Core 1 (affinity mask = 3)
				// Guard: Only apply if system has more than 2 logical cores
				if sysAff > 3 {
					_ = setProcessAffinity(pid, 3)
				}
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
	return nil
}

// IsolateGameProcess binds the game exclusively to Cores 2 to N, bypassing background noise.
// It returns the original game process affinity mask to allow precise restoration.
func IsolateGameProcess(gamePID uint32) (uintptr, error) {
	procAff, sysAff, err := getProcessAffinity(gamePID)
	if err != nil {
		return 0, fmt.Errorf("failed to read game process affinity: %w", err)
	}

	// gameMask clears the first two bits (Core 0 and Core 1)
	gameMask := sysAff &^ uintptr(3)

	// Fallback: If system is dual-core or less, keep original affinity
	if sysAff <= 3 {
		gameMask = procAff
	}

	err = setProcessAffinity(gamePID, gameMask)
	if err != nil {
		return 0, fmt.Errorf("failed to apply CPU affinity to game: %w", err)
	}

	return procAff, nil
}

// RestoreAffinities restores all background and game processes back to their original core maps.
func RestoreAffinities(gamePID uint32, originalGameMask uintptr) {
	affinityMutex.Lock()
	defer affinityMutex.Unlock()

	// 1. Restore background processes
	for pid, originalMask := range originalBackgroundAffinities {
		_ = setProcessAffinity(pid, originalMask)
		delete(originalBackgroundAffinities, pid)
	}

	// 2. Restore game process
	if originalGameMask != 0 {
		_ = setProcessAffinity(gamePID, originalGameMask)
	}
}
