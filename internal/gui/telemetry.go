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

	"nosboost/internal/config"
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
					hudDetails.SetText(config.T("hud_safe_default"))
					netStatus.SetText(config.T("net_state_default"))
					packetShield.SetText(config.T("packet_inactive"))
				} else if mode == "Balanced" {
					hudDetails.SetText(config.T("hud_balanced"))
					netStatus.SetText(config.T("net_state_opt"))
					packetShield.SetText(config.T("packet_active"))
				} else {
					hudDetails.SetText(config.T("hud_extreme"))
					netStatus.SetText(config.T("net_state_opt"))
					packetShield.SetText(config.T("packet_active"))
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
			ramAlloc.SetText(config.T("srv_query_failed"))
		})
		return
	}

	totalGB := float64(mem.TotalPhys) / (1024 * 1024 * 1024)
	availGB := float64(mem.AvailPhys) / (1024 * 1024 * 1024)
	allocGB := totalGB - availGB
	percent := float64(mem.MemoryLoad)

	fyne.Do(func() {
		ramAlloc.SetText(fmt.Sprintf(config.T("allocated_ram"), fmt.Sprintf("%.2f GB / %.2f GB (%d%%)", allocGB, totalGB, int(percent))))
		memProgress.SetValue(percent / 100.0)
	})
}

// updateSchedulerTelemetry displays scheduler, timer, HAGS, and Game Mode states
func updateSchedulerTelemetry(cpuOpt, timer, hags, gameMode *widget.Label) {
	mode := GetCurrentMode()

	var cpuOptText, timerText string
	if mode == "Safe Default" {
		cpuOptText = config.T("cpu_state_default")
		timerText = config.T("timer_state_default")
	} else {
		cpuOptText = config.T("cpu_state_opt")
		timerText = config.T("timer_state_opt")
	}

	hagsVal, hagsErr := syswatch.GetHAGSState()
	var hagsText string
	if hagsErr != nil {
		hagsText = config.T("hags_error")
	} else if hagsVal == 2 {
		hagsText = config.T("hags_enabled")
	} else if hagsVal == 1 {
		hagsText = config.T("hags_disabled")
	} else {
		hagsText = config.T("hags_unsupported")
	}

	gmActive, gmErr := syswatch.GetGameModeState()
	var gmText string
	if gmErr != nil {
		gmText = config.T("game_mode_error")
	} else if gmActive {
		gmText = config.T("game_mode_enabled")
	} else {
		gmText = config.T("game_mode_disabled")
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
			srvStatus.SetText(config.T("srv_query_failed"))
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
			srvStatus.SetText(config.T("srv_state_opt"))
		} else if updatesStopped || telemetryStopped || sysmainStopped {
			srvStatus.SetText(config.T("srv_state_partial"))
		} else {
			srvStatus.SetText(config.T("srv_state_default"))
		}
	})
}
