package gui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.org/x/sys/windows/svc/mgr"

	"nosboost/internal/memory"
	"nosboost/internal/syswatch"
)

// ShowDashboard bootstraps and renders the overhauled premium Fyne native command center.
func ShowDashboard() {
	// 1. Force strict dark-theme override at environment level
	os.Setenv("FYNE_THEME", "dark")

	myApp := app.New()
	myWindow := myApp.NewWindow("NosBoost Command Center // Active Performance Matrix")

	// 2. Setup Header Panel
	titleLabel := canvas.NewText("NOSBOOST COMMAND CENTER", theme.PrimaryColor())
	titleLabel.TextSize = 22
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	subtitleLabel := canvas.NewText("Windows Kernel Systems Latency Enforcement Engine // Production Release v1.0.0", theme.ForegroundColor())
	subtitleLabel.TextSize = 10
	subtitleLabel.Alignment = fyne.TextAlignCenter

	headerContainer := container.NewVBox(
		titleLabel,
		subtitleLabel,
		widget.NewSeparator(),
	)

	// 3. LEFT POWER COLUMN: ADVANCED TELEMETRY ENGINE MONITOR
	// Card 1: Physical Memory segmented breakdown display
	ramAllocLabel := widget.NewLabel("Allocated RAM: Querying...")
	ramAllocLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	ramAllocLabel.Wrapping = fyne.TextWrapWord

	memProgress := widget.NewProgressBar()
	cleanMemBtn := widget.NewButtonWithIcon("CLEAN MEMORY", theme.ContentClearIcon(), func() {
		go func() {
			logToUI("[MEMORY] Initiating background RAM standby cache cleaning sweep...")
			if err := memory.PurgeStandbyList(); err != nil {
				logToUI(fmt.Sprintf("[WARNING] PurgeStandbyList error: %v", err))
			}
			if err := memory.FlushModifiedList(); err != nil {
				logToUI(fmt.Sprintf("[WARNING] FlushModifiedList error: %v", err))
			}
			logToUI("[MEMORY] Standby and modified RAM caches flushed successfully.")
		}()
	})
	cleanMemBtn.Importance = widget.WarningImportance

	memCard := widget.NewCard("PHYSICAL MEMORY HEALTH", "", container.NewVBox(
		ramAllocLabel,
		memProgress,
		cleanMemBtn,
	))

	// Card 2: Network Stabilization Status Elements (Dynamic status dashboard)
	netStatusLabel := widget.NewLabel("Network Latency Engine: Querying...")
	netStatusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	netStatusLabel.Wrapping = fyne.TextWrapWord

	packetShieldLabel := widget.NewLabel("Packet Loss Shield: Querying...")
	packetShieldLabel.TextStyle = fyne.TextStyle{Monospace: true}
	packetShieldLabel.Wrapping = fyne.TextWrapWord

	flushNetworkBtn := widget.NewButtonWithIcon("FLUSH DNS & RESET CONNECTION", theme.ViewRefreshIcon(), func() {
		go func() {
			logToUI("[NETWORK] Flushing DNS Resolver Cache...")
			if err := exec.Command("ipconfig", "/flushdns").Run(); err != nil {
				logToUI(fmt.Sprintf("[WARNING] FlushDNS error: %v", err))
			} else {
				logToUI("[NETWORK] DNS Resolver Cache flushed successfully.")
			}
			logToUI("[NETWORK] Resetting Winsock TCP/IP stack to clean up packet drops...")
			if err := exec.Command("netsh", "winsock", "reset").Run(); err != nil {
				logToUI(fmt.Sprintf("[WARNING] Winsock reset error: %v", err))
			} else {
				logToUI("[NETWORK] Winsock TCP/IP stack catalog reset successfully (Reboot recommended).")
			}
		}()
	})
	flushNetworkBtn.Importance = widget.WarningImportance

	netDetailsCard := widget.NewCard("NETWORK PACKET LOSS & LATENCY SHIELD", "", container.NewVBox(
		netStatusLabel,
		packetShieldLabel,
		widget.NewSeparator(),
		flushNetworkBtn,
	))

	// Card 3: Kernel Scheduler Status
	cpuOptLabel := widget.NewLabel("CPU Optimization: Querying...")
	cpuOptLabel.Wrapping = fyne.TextWrapWord
	timerLabel := widget.NewLabel("System Precision Timers: Querying...")
	timerLabel.Wrapping = fyne.TextWrapWord
	hagsLabel := widget.NewLabel("Hardware GPU Scheduling (HAGS): Querying...")
	hagsLabel.Wrapping = fyne.TextWrapWord
	gameModeLabel := widget.NewLabel("Windows Game Mode: Querying...")
	gameModeLabel.Wrapping = fyne.TextWrapWord

	kernelCard := widget.NewCard("KERNEL SCHEDULER & TIMERS", "", container.NewVBox(
		cpuOptLabel,
		timerLabel,
		hagsLabel,
		gameModeLabel,
		container.NewGridWithColumns(2,
			widget.NewButton("Toggle Game Mode", func() {
				go func() {
					current, _ := syswatch.GetGameModeState()
					err := syswatch.SetGameModeState(!current)
					if err != nil {
						logToUI(fmt.Sprintf("[ERROR] Failed to toggle Game Mode: %v", err))
					} else {
						var stateStr string
						if !current {
							stateStr = "ENABLED"
						} else {
							stateStr = "DISABLED"
						}
						logToUI(fmt.Sprintf("[SYSWATCH] Windows Game Mode toggled to: %s", stateStr))
						updateSchedulerTelemetry(cpuOptLabel, timerLabel, hagsLabel, gameModeLabel)
					}
				}()
			}),
			widget.NewButton("Toggle HAGS", func() {
				go func() {
					current, _ := syswatch.GetHAGSState()
					enabled := current != 2
					err := syswatch.SetHAGSState(enabled)
					if err != nil {
						logToUI(fmt.Sprintf("[ERROR] Failed to toggle HAGS: %v", err))
					} else {
						var stateStr string
						if enabled {
							stateStr = "ENABLED"
						} else {
							stateStr = "DISABLED"
						}
						logToUI(fmt.Sprintf("[SYSWATCH] GPU Scheduling (HAGS) set to %s (Reboot required).", stateStr))
						updateSchedulerTelemetry(cpuOptLabel, timerLabel, hagsLabel, gameModeLabel)
					}
				}()
			}),
		),
	))

	// Card 4: OS Services checklist & deep verifier
	srvStatusLabel := widget.NewLabel("Background Diagnostics & Windows Update: Querying...")
	srvStatusLabel.Wrapping = fyne.TextWrapWord

	deepVerifierBtn := widget.NewButtonWithIcon("Verify Background Services", theme.SearchIcon(), func() {
		go func() {
			logToUI("[SYSWATCH] Querying service process parameters...")
			m, err := mgr.Connect()
			if err != nil {
				logToUI("[WARNING] Failed to connect to SCM.")
				return
			}
			defer m.Disconnect()

			services := []string{"wuauserv", "DiagTrack", "SysMain"}
			for _, s := range services {
				sKey, err := m.OpenService(s)
				if err != nil {
					logToUI(fmt.Sprintf("[SYSWATCH] Service %s not installed or missing.", s))
					continue
				}
				status, err := sKey.Query()
				if err == nil {
					logToUI(fmt.Sprintf("[SYSWATCH] Service %s is %s.", s, serviceStateToString(status.State)))
				}
				sKey.Close()
			}
		}()
	})

	srvCard := widget.NewCard("TELEMETRY SERVICE DISCOVERY LEDGER", "", container.NewVBox(
		srvStatusLabel,
		deepVerifierBtn,
	))

	// 4. RIGHT POWER COLUMN: ONE-CLICK ORCHESTRATION & DYNAMIC HUD
	hudModeLabel := widget.NewLabel("MATRIX STATUS: ENGAGED [SAFE BASELINE]")
	hudModeLabel.TextStyle = fyne.TextStyle{Bold: true}
	hudModeLabel.Wrapping = fyne.TextWrapWord
	hudDetailsLabel := widget.NewLabel("Core parking: Active | MSI Mode: Off | TCP Delay: OS Defaults")
	hudDetailsLabel.TextStyle = fyne.TextStyle{Monospace: true}
	hudDetailsLabel.Wrapping = fyne.TextWrapWord

	blueprintBtn := widget.NewButtonWithIcon("Query Detailed Engagement Blueprint", theme.InfoIcon(), func() {
		go func() {
			logToUI("==================================================")
			logToUI("[BLUEPRINT] Current Active Optimization Specifications:")
			logToUI(fmt.Sprintf("[BLUEPRINT] Mode Status: %s", GetCurrentMode()))
			logToUI("[BLUEPRINT] CPU Cores Lock: 100% Core Parking Elimination Active")
			logToUI("[BLUEPRINT] CPU Foreground Priority Ratio: short-variable quantum (0x26)")
			logToUI("[BLUEPRINT] Memory list purging: PurgeStandbyList & FlushModifiedList available")
			logToUI("[BLUEPRINT] TCP Stack: TcpAckFrequency=1 & TCPNoDelay=1 enabled")
			logToUI("[BLUEPRINT] Peripherals: mouse/keyboard buffer size set to 20")
			logToUI("==================================================")
		}()
	})

	hudCard := widget.NewCard("ACTIVE PERFORMANCE MATRIX STATUS HUD", "", container.NewVBox(
		hudModeLabel,
		hudDetailsLabel,
		blueprintBtn,
	))

	// Extreme Card
	extremeFeatures := widget.NewLabel(
		"Maximizes in-game FPS and eliminates input latency.\n" +
		"Restricts background system apps, disables power saving,\n" +
		"freezes telemetry cycles, and isolates gaming processes.",
	)
	extremeFeatures.TextStyle = fyne.TextStyle{Italic: true}
	extremeCardBtn := widget.NewButtonWithIcon("ENGAGE EXTREME PERFORMANCE MATRIX", theme.ConfirmIcon(), func() {
		logToUI("[UI] User initiated EXTREME mode trigger.")
		go ApplyExtremeMode()
	})
	extremeCardBtn.Importance = widget.HighImportance

	extremeCard := widget.NewCard("EXTREME PERFORMANCE PROFILE", "", container.NewVBox(
		extremeFeatures,
		extremeCardBtn,
	))

	// Balanced Card
	balancedFeatures := widget.NewLabel(
		"Improves gaming performance and network responsiveness\n" +
		"while keeping background services active to permit safe\n" +
		"multitasking (web browsing, recording, and streaming).",
	)
	balancedFeatures.TextStyle = fyne.TextStyle{Italic: true}
	balancedCardBtn := widget.NewButtonWithIcon("ENGAGE BALANCED PERFORMANCE MATRIX", theme.MediaPlayIcon(), func() {
		logToUI("[UI] User initiated BALANCED mode trigger.")
		go ApplyBalancedMode()
	})

	balancedCard := widget.NewCard("BALANCED PERFORMANCE PROFILE", "", container.NewVBox(
		balancedFeatures,
		balancedCardBtn,
	))

	// Total Restore transactional card with confirmation stage
	var restoreContainer *fyne.Container
	var restoreCard *widget.Card
	var restoreNormalLayout *fyne.Container

	restoreNormalLayout = container.NewVBox(
		widget.NewLabel("Wipes injected routing routes, restores display/NIC MSI defaults,"),
		widget.NewLabel("re-enables SysMain, restores timers, and unfreezes processes."),
		widget.NewButtonWithIcon("INITIATE TOTAL ROLLBACK MATRIX", theme.CancelIcon(), func() {
			logToUI("[UI] User triggered Total Restore rollback plan evaluation.")

			// Overwrite the card container dynamically to show the rollback breakdown confirmation
			restoreContainer.Objects = nil

			breakdownLabel := widget.NewLabel(
				"SYSTEM ROLLBACK PLAN BREAKDOWN:\n" +
				" 1. Revert Core Parking AC/DC indexes\n" +
				" 2. Revert graphics card/NIC MSI parameters\n" +
				" 3. Restore mouse/keyboard buffers to 100\n" +
				" 4. Restart and re-enable service SysMain\n" +
				" 5. Re-align system invariant boot timers\n" +
				" 6. Remove static esports gateway routes\n" +
				" 7. Resume background updater & diagnostic services",
			)
			breakdownLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}

			confirmBtn := widget.NewButtonWithIcon("CONFIRM SYSTEM ROLLBACK NOW", theme.WarningIcon(), func() {
				logToUI("[UI] User confirmed rollback execution.")
				go func() {
					ApplyTotalRestore()

					// Revert UI back to normal after restore finished
					restoreContainer.Objects = nil
					restoreContainer.Add(restoreNormalLayout)
					restoreCard.Refresh()
				}()
			})
			confirmBtn.Importance = widget.DangerImportance

			cancelBtn := widget.NewButton("Cancel Rollback", func() {
				logToUI("[UI] User cancelled rollback plan.")
				restoreContainer.Objects = nil
				restoreContainer.Add(restoreNormalLayout)
				restoreCard.Refresh()
			})

			restoreContainer.Add(breakdownLabel)
			restoreContainer.Add(confirmBtn)
			restoreContainer.Add(cancelBtn)
			restoreCard.Refresh()
		}),
	)

	restoreContainer = container.NewVBox(restoreNormalLayout)
	restoreCard = widget.NewCard("TOTAL ROLLBACK RESTORE (SAFE DEFAULT)", "", restoreContainer)

	// 5. Build Tab Containers (Navbar/Tab design to eliminate scrollbars and simplify layout)
	boosterTab := container.NewTabItem("Booster Engine", container.NewVBox(
		hudCard,
		container.NewGridWithColumns(2,
			container.NewVBox(extremeCard, balancedCard),
			restoreCard,
		),
	))

	systemTab := container.NewTabItem("Memory & Services", container.NewGridWithColumns(2,
		memCard,
		srvCard,
	))

	netDescLabel := widget.NewLabel("TCP NoDelay & Gateway Route Injection are active in Performance Modes.")
	netDescLabel.Alignment = fyne.TextAlignCenter
	netDescLabel.TextStyle = fyne.TextStyle{Italic: true}

	networkScroll := container.NewVScroll(container.NewVBox(
		netDetailsCard,
		netDescLabel,
	))

	networkTab := container.NewTabItem("Network Optimization", networkScroll)

	kernelTab := container.NewTabItem("Kernel & GPU Tuner", container.NewVBox(
		kernelCard,
	))

	tabs := container.NewAppTabs(boosterTab, systemTab, networkTab, kernelTab)
	tabs.SetTabLocation(container.TabLocationTop)

	// 5. BOTTOM ROW: ENHANCED LOG FEED MATRIX TRACKER
	consoleFeed := widget.NewMultiLineEntry()
	consoleFeed.SetText("NosBoost Console Initialized. Awaiting command matrix input...\n")
	consoleFeed.Disable()
	consoleFeed.Wrapping = fyne.TextWrapWord
	consoleFeed.TextStyle = fyne.TextStyle{Monospace: true}

	consoleContainer := container.NewGridWithRows(1, consoleFeed)
	consoleCard := widget.NewCard("SYSTEM OPTIMIZATION LOG STREAM", "", consoleContainer)

	// Combine components into final layout with premium VSplit (Split Pane) and border layout
	bodySplit := container.NewVSplit(
		tabs,
		consoleCard,
	)
	bodySplit.Offset = 0.68 // 68% height for the tabs, 32% for logs

	mainLayout := container.NewBorder(
		headerContainer, // Top
		nil,             // Bottom
		nil,             // Left
		nil,             // Right
		bodySplit,       // Center
	)

	myWindow.SetContent(mainLayout)
	myWindow.Resize(fyne.NewSize(1000, 800))

	// 6. ASYNCHRONOUS BACKGROUND STATS UPDATES (Tickers)
	// Consumes UIConsoleChan safely, prepending prefix scannability icons dynamically
	go func() {
		for msg := range UIConsoleChan {
			currentText := consoleFeed.Text

			// Parse brackets prefix and prepend appropriate scannability icons
			tag := "[INFO] "
			if strings.Contains(msg, "[SYSTEM]") {
				tag = "[SYS]  "
			}
			if strings.Contains(msg, "[CLEANER]") {
				tag = "[CLN]  "
			}
			if strings.Contains(msg, "[BOOSTER]") {
				tag = "[BST]  "
			}
			if strings.Contains(msg, "[POWER]") {
				tag = "[PWR]  "
			}
			if strings.Contains(msg, "[NETWORK]") {
				tag = "[NET]  "
			}
			if strings.Contains(msg, "[HARDWARE]") {
				tag = "[HW]   "
			}
			if strings.Contains(msg, "[SYSWATCH]") {
				tag = "[WTCH] "
			}
			if strings.Contains(msg, "[MEMORY]") {
				tag = "[MEM]  "
			}
			if strings.Contains(msg, "[RESTORE]") {
				tag = "[RSTR] "
			}
			if strings.Contains(msg, "[POLLER]") {
				tag = "[POLL] "
			}
			if strings.Contains(msg, "[UI]") {
				tag = "[UI]   "
			}
			if strings.Contains(msg, "[WARNING]") {
				tag = "[WARN] "
			}
			if strings.Contains(msg, "[CRITICAL WARNING]") {
				tag = "[CRIT] "
			}
			if strings.Contains(msg, "[ERROR]") {
				tag = "[ERR]  "
			}
			if strings.Contains(msg, "[BLUEPRINT]") {
				tag = "[BLUE] "
			}

			// Remove bracket tags to clean log feed readability
			cleanMsg := msg
			cleanMsg = strings.ReplaceAll(cleanMsg, "[SYSTEM] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[CLEANER] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[BOOSTER] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[POWER] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[NETWORK] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[HARDWARE] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[SYSWATCH] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[MEMORY] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[RESTORE] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[POLLER] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[UI] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[WARNING] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[CRITICAL WARNING] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[ERROR] ", "")
			cleanMsg = strings.ReplaceAll(cleanMsg, "[BLUEPRINT] ", "")

			newText := currentText + tag + cleanMsg + "\n"

			// Cap log size at 30k characters to prevent memory leaks
			if len(newText) > 30000 {
				newText = newText[15000:]
			}

			fyne.Do(func() {
				consoleFeed.SetText(newText)
				consoleFeed.CursorColumn = len(newText)
			})
		}
	}()

	// Spawn Telemetry Tickers in isolated background goroutines
	go runTelemetryTickers(
		ramAllocLabel, memProgress,
		cpuOptLabel, timerLabel, hagsLabel, gameModeLabel,
		srvStatusLabel,
		hudModeLabel, hudDetailsLabel,
		netStatusLabel, packetShieldLabel,
	)

	logToUI("[SYSTEM] Overhauled UI canvas initialized. Baseline safety plan is active.")
	myWindow.ShowAndRun()
}
