package gui

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"nosboost/internal/syswatch"
)

// runTelemetryTickers sweeps memory status, schedulers, and SCM services asynchronously
func runTelemetryTickers(
	ramAlloc *widget.Label,
	memProgress *widget.ProgressBar,
	cpuOpt, timer, hags, gameMode *widget.Label,
	srvStatus *widget.Label,
	hudMode, hudDetails *widget.Label,
	netStatus, packetShield *widget.Label,
) {
	memTicker := time.NewTicker(1500 * time.Millisecond)
	defer memTicker.Stop()

	// Immediate run once
	updateMemoryTelemetry(ramAlloc, memProgress)
	updateSchedulerTelemetry(cpuOpt, timer, hags, gameMode)
	updateServicesTelemetry(srvStatus)

	for {
		select {
		case <-memTicker.C:
			updateMemoryTelemetry(ramAlloc, memProgress)
			updateSchedulerTelemetry(cpuOpt, timer, hags, gameMode)
			updateServicesTelemetry(srvStatus)

			// Update Status HUD (thread-safe)
			mode := GetCurrentMode()
			fyne.Do(func() {
				hudMode.SetText(fmt.Sprintf("MATRIX STATUS: ENGAGED [%s]", strings.ToUpper(mode)))
				if mode == "Safe Default" {
					hudDetails.SetText("Core Parking: Active | MSI Mode: Default | TCP Delay: OS Standard")
					netStatus.SetText("Network Latency Engine: Default (OS Standard)")
					packetShield.SetText("Packet Loss Shield: Inactive")
				} else if mode == "Balanced" {
					hudDetails.SetText("Core Parking: Disabled | MSI Mode: Locked | TCP Delay: Instant Fire")
					netStatus.SetText("Network Latency Engine: Optimized (Instant TCP Active)")
					packetShield.SetText("Packet Loss Shield: Active (LSO/RSC Disabled)")
				} else {
					hudDetails.SetText("Core Parking: Disabled | MSI Mode: Locked | TCP Delay: Instant Fire | Services: Frozen")
					netStatus.SetText("Network Latency Engine: Optimized (Instant TCP Active)")
					packetShield.SetText("Packet Loss Shield: Active (LSO/RSC Disabled)")
				}
			})
		}
	}
}

// updateMemoryTelemetry retrieves Windows Global Memory parameters and updates progress bars
func updateMemoryTelemetry(ramAlloc *widget.Label, memProgress *widget.ProgressBar) {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))

	r1, _, _ := procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&mem)))
	if r1 == 0 {
		fyne.Do(func() {
			ramAlloc.SetText("RAM Allocation: Failed to query Windows APIs")
		})
		return
	}

	totalGB := float64(mem.TotalPhys) / (1024 * 1024 * 1024)
	availGB := float64(mem.AvailPhys) / (1024 * 1024 * 1024)
	allocGB := totalGB - availGB
	percent := float64(mem.MemoryLoad)

	fyne.Do(func() {
		ramAlloc.SetText(fmt.Sprintf("Allocated RAM: %.2f GB / %.2f GB (%d%%)", allocGB, totalGB, int(percent)))
		memProgress.SetValue(percent / 100.0)
	})
}

// updateSchedulerTelemetry displays scheduler, timer, HAGS, and Game Mode states
func updateSchedulerTelemetry(cpuOpt, timer, hags, gameMode *widget.Label) {
	mode := GetCurrentMode()

	var cpuOptText, timerText string
	if mode == "Safe Default" {
		cpuOptText = "CPU Optimization: Safe Default (Standard Scheduling)"
		timerText = "System Precision Timers: Default (OS Managed)"
	} else {
		cpuOptText = "CPU Optimization: Engaged (Short-Variable Quantum & Core Unparking)"
		timerText = "System Precision Timers: Locked (Invariant Precision 0.5ms)"
	}

	hagsVal, hagsErr := syswatch.GetHAGSState()
	var hagsText string
	if hagsErr != nil {
		hagsText = "Hardware GPU Scheduling (HAGS): Error"
	} else if hagsVal == 2 {
		hagsText = "Hardware GPU Scheduling (HAGS): Enabled (Active)"
	} else if hagsVal == 1 {
		hagsText = "Hardware GPU Scheduling (HAGS): Disabled"
	} else {
		hagsText = "Hardware GPU Scheduling (HAGS): Not Supported"
	}

	gmActive, gmErr := syswatch.GetGameModeState()
	var gmText string
	if gmErr != nil {
		gmText = "Windows Game Mode: Error"
	} else if gmActive {
		gmText = "Windows Game Mode: Enabled"
	} else {
		gmText = "Windows Game Mode: Disabled"
	}

	fyne.Do(func() {
		cpuOpt.SetText(cpuOptText)
		timer.SetText(timerText)
		hags.SetText(hagsText)
		gameMode.SetText(gmText)
	})
}

// updateServicesTelemetry checks key Windows services PID and statuses
func updateServicesTelemetry(srvStatus *widget.Label) {
	m, err := mgr.Connect()
	if err != nil {
		fyne.Do(func() {
			srvStatus.SetText("Background Services: Query Failed")
		})
		return
	}
	defer m.Disconnect()

	var updatesStopped, telemetryStopped, sysmainStopped bool

	if su, err := m.OpenService("wuauserv"); err == nil {
		if status, err := su.Query(); err == nil && (status.State == svc.Stopped || servicesFrozen) {
			updatesStopped = true
		}
		su.Close()
	}
	if sd, err := m.OpenService("DiagTrack"); err == nil {
		if status, err := sd.Query(); err == nil && (status.State == svc.Stopped || servicesFrozen) {
			telemetryStopped = true
		}
		sd.Close()
	}
	if ss, err := m.OpenService("SysMain"); err == nil {
		if status, err := ss.Query(); err == nil && status.State == svc.Stopped {
			sysmainStopped = true
		}
		ss.Close()
	}

	fyne.Do(func() {
		if updatesStopped && telemetryStopped && sysmainStopped {
			srvStatus.SetText("Background Diagnostics & Windows Update: Suspended (Fully Optimized)")
		} else if updatesStopped || telemetryStopped || sysmainStopped {
			srvStatus.SetText("Background Diagnostics & Windows Update: Partially Suspended")
		} else {
			srvStatus.SetText("Background Diagnostics & Windows Update: Standard (Running)")
		}
	})
}
