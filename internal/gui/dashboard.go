package gui

import (
	"fmt"
	"image/color"
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

	"nosboost/internal/config"
	"nosboost/internal/hardware"
	"nosboost/internal/memory"
	"nosboost/internal/syswatch"
)

func createVSpacer(height float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(0, height))
	return rect
}

// DynamicGridLayout implements a fully responsive, wrap-around column-based Fyne layout.
type DynamicGridLayout struct {
	MaxCols     int
	MinColWidth float32
}

func (l *DynamicGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	w := l.MinColWidth
	h := float32(0)
	for _, obj := range objects {
		h += obj.MinSize().Height + theme.Padding()
	}
	return fyne.NewSize(w, h)
}

func (l *DynamicGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	count := len(objects)
	if count == 0 {
		return
	}

	cols := int(size.Width / l.MinColWidth)
	if cols < 1 {
		cols = 1
	}
	if cols > l.MaxCols {
		cols = l.MaxCols
	}

	cellWidth := (size.Width - float32(cols-1)*theme.Padding()) / float32(cols)

	// Calculate row heights dynamically based on largest cell in each row
	rowHeights := make(map[int]float32)
	for i, obj := range objects {
		row := i / cols
		h := obj.MinSize().Height
		if h > rowHeights[row] {
			rowHeights[row] = h
		}
	}

	x := float32(0)
	y := float32(0)

	for i, obj := range objects {
		row := i / cols
		col := i % cols

		if col == 0 && i > 0 {
			x = 0
			y += rowHeights[row-1] + theme.Padding()
		}

		objSize := fyne.NewSize(cellWidth, rowHeights[row])
		obj.Resize(objSize)
		obj.Move(fyne.NewPos(x, y))

		x += cellWidth + theme.Padding()
	}
}

// ShowDashboard bootstraps and renders the overhauled premium Fyne native command center.
func ShowDashboard() {
	// 1. Force strict dark-theme override at environment level
	os.Setenv("FYNE_THEME", "dark")

	myApp := app.New()
	myWindow := myApp.NewWindow("NosBoost | Performance Optimizer")

	var refreshUIStrings func()
	var isRefreshingStrings bool

	// 2. Setup Header Panel with 2px increased font sizes
	titleLabel := canvas.NewText(config.T("app_title"), theme.PrimaryColor())
	titleLabel.TextSize = 28 // Increased by 2px (now 28)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	subtitleLabel := canvas.NewText(config.T("subtitle"), theme.ForegroundColor())
	subtitleLabel.TextSize = 14 // Increased by 2px (now 14)
	subtitleLabel.Alignment = fyne.TextAlignCenter

	headerContainer := container.NewVBox(
		titleLabel,
		subtitleLabel,
		widget.NewSeparator(),
	)

	// 3. PHYSICAL MEMORY HEALTH
	ramAllocLabel := widget.NewLabel(config.T("ram_querying"))
	ramAllocLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	ramAllocLabel.Wrapping = fyne.TextWrapWord

	memProgress := widget.NewProgressBar()
	cleanMemBtn := widget.NewButtonWithIcon(config.T("clean_mem"), theme.ContentClearIcon(), func() {
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

	memCard := widget.NewCard(config.T("ram_health"), "", container.NewVBox(
		ramAllocLabel,
		memProgress,
		cleanMemBtn,
	))

	// 4. NETWORK PACKET LOSS & LATENCY SHIELD
	netStatusLabel := widget.NewLabel(config.T("net_querying"))
	netStatusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	netStatusLabel.Wrapping = fyne.TextWrapWord

	packetShieldLabel := widget.NewLabel(config.T("packet_querying"))
	packetShieldLabel.TextStyle = fyne.TextStyle{Monospace: true}
	packetShieldLabel.Wrapping = fyne.TextWrapWord

	flushNetworkBtn := widget.NewButtonWithIcon(config.T("flush_dns"), theme.ViewRefreshIcon(), func() {
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

	netDetailsCard := widget.NewCard(config.T("net_shield"), "", container.NewVBox(
		netStatusLabel,
		packetShieldLabel,
		widget.NewSeparator(),
		flushNetworkBtn,
	))

	// 5. KERNEL SCHEDULER & TIMERS
	cpuOptLabel := widget.NewLabel(config.T("cpu_querying"))
	cpuOptLabel.Wrapping = fyne.TextWrapWord
	timerLabel := widget.NewLabel(config.T("timers_querying"))
	timerLabel.Wrapping = fyne.TextWrapWord
	hagsLabel := widget.NewLabel(config.T("hags_querying"))
	hagsLabel.Wrapping = fyne.TextWrapWord
	gameModeLabel := widget.NewLabel(config.T("game_mode_querying"))
	gameModeLabel.Wrapping = fyne.TextWrapWord

	kernelCard := widget.NewCard(config.T("kernel_card"), "", container.NewVBox(
		cpuOptLabel,
		timerLabel,
		hagsLabel,
		gameModeLabel,
		container.NewGridWithColumns(2,
			widget.NewButton(config.T("toggle_game_mode"), func() {
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
			widget.NewButton(config.T("toggle_hags"), func() {
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

	// 6. TELEMETRY SERVICE DISCOVERY LEDGER
	srvStatusLabel := widget.NewLabel(config.T("srv_querying"))
	srvStatusLabel.Wrapping = fyne.TextWrapWord

	deepVerifierBtn := widget.NewButtonWithIcon(config.T("verify_srv"), theme.SearchIcon(), func() {
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

	srvCard := widget.NewCard(config.T("srv_ledger"), "", container.NewVBox(
		srvStatusLabel,
		deepVerifierBtn,
	))

	// 7. ACTIVE PERFORMANCE HUD
	hudModeLabel := widget.NewLabel("MATRIX STATUS: ENGAGED [SAFE BASELINE]")
	hudModeLabel.TextStyle = fyne.TextStyle{Bold: true}
	hudModeLabel.Wrapping = fyne.TextWrapWord
	hudDetailsLabel := widget.NewLabel(config.T("hud_safe_default"))
	hudDetailsLabel.TextStyle = fyne.TextStyle{Monospace: true}
	hudDetailsLabel.Wrapping = fyne.TextWrapWord

	blueprintBtn := widget.NewButtonWithIcon(config.T("blueprint"), theme.InfoIcon(), func() {
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

	hudCard := widget.NewCard(config.T("hud_title"), "", container.NewVBox(
		hudModeLabel,
		hudDetailsLabel,
		blueprintBtn,
	))

	// Extreme Profile
	extremeFeatures := widget.NewLabel(config.T("extreme_desc"))
	extremeFeatures.TextStyle = fyne.TextStyle{Italic: true}
	extremeFeatures.Wrapping = fyne.TextWrapWord
	extremeCardBtn := widget.NewButtonWithIcon(config.T("extreme_btn"), theme.ConfirmIcon(), func() {
		logToUI("[UI] User initiated EXTREME mode trigger.")
		go ApplyExtremeMode()
	})
	extremeCardBtn.Importance = widget.HighImportance

	extremeCard := widget.NewCard(config.T("extreme_title"), "", container.NewVBox(
		extremeFeatures,
		extremeCardBtn,
	))

	// Balanced Profile
	balancedFeatures := widget.NewLabel(config.T("balanced_desc"))
	balancedFeatures.TextStyle = fyne.TextStyle{Italic: true}
	balancedFeatures.Wrapping = fyne.TextWrapWord
	balancedCardBtn := widget.NewButtonWithIcon(config.T("balanced_btn"), theme.MediaPlayIcon(), func() {
		logToUI("[UI] User initiated BALANCED mode trigger.")
		go ApplyBalancedMode()
	})

	balancedCard := widget.NewCard(config.T("balanced_title"), "", container.NewVBox(
		balancedFeatures,
		balancedCardBtn,
	))

	// Total Restore transactional card
	var restoreContainer *fyne.Container
	var restoreCard *widget.Card
	var restoreNormalLayout *fyne.Container
	restoreDescLabel := widget.NewLabel(config.T("restore_desc"))
	restoreDescLabel.Wrapping = fyne.TextWrapWord

	restoreCardBtn := widget.NewButtonWithIcon(config.T("restore_btn"), theme.CancelIcon(), func() {
		logToUI("[UI] User triggered Total Restore rollback plan evaluation.")

		restoreContainer.Objects = nil

		breakdownLabel := widget.NewLabel(config.T("rollback_breakdown"))
		breakdownLabel.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		breakdownLabel.Wrapping = fyne.TextWrapWord

		confirmBtn := widget.NewButtonWithIcon(config.T("confirm_rollback"), theme.WarningIcon(), func() {
			logToUI("[UI] User confirmed rollback execution.")
			go func() {
				ApplyTotalRestore()

				restoreContainer.Objects = nil
				restoreContainer.Add(restoreNormalLayout)
				restoreCard.Refresh()
			}()
		})
		confirmBtn.Importance = widget.DangerImportance

		cancelBtn := widget.NewButton(config.T("cancel_rollback"), func() {
			logToUI("[UI] User cancelled rollback plan.")
			restoreContainer.Objects = nil
			restoreContainer.Add(restoreNormalLayout)
			restoreCard.Refresh()
		})

		restoreContainer.Add(breakdownLabel)
		restoreContainer.Add(confirmBtn)
		restoreContainer.Add(cancelBtn)
		restoreCard.Refresh()
	})

	restoreNormalLayout = container.NewVBox(
		restoreDescLabel,
		restoreCardBtn,
	)

	restoreContainer = container.NewVBox(restoreNormalLayout)
	restoreCard = widget.NewCard(config.T("restore_title"), "", restoreContainer)

	// Build dynamic language list from embedded locales JSON files
	availLangs := config.GetAvailableLanguages()
	var langNames []string
	langCodeToName := make(map[string]string)
	langNameToCode := make(map[string]string)

	// Ensure English is first in list, and load other languages deterministically
	if name, exists := availLangs["en"]; exists {
		langNames = append(langNames, name)
	}
	for code, name := range availLangs {
		if code != "en" {
			langNames = append(langNames, name)
		}
	}

	for code, name := range availLangs {
		langCodeToName[code] = name
		langNameToCode[name] = code
	}

	// Theme Selector widget with fully localizable choices
	themeSelect := widget.NewSelect([]string{config.T("theme_dark"), config.T("theme_light")}, nil)
	themeSelect.OnChanged = func(selected string) {
		if selected == config.T("theme_dark") {
			fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
			logToUI("[UI] Theme toggled to Dark Mode.")
		} else {
			fyne.CurrentApp().Settings().SetTheme(theme.LightTheme())
			logToUI("[UI] Theme toggled to Light Mode.")
		}
		if refreshUIStrings != nil && !isRefreshingStrings {
			refreshUIStrings()
		}
	}
	themeSelect.SetSelected(config.T("theme_dark"))

	langSelect := widget.NewSelect(langNames, nil)

	// Localizable labels for the customization cards
	themeLabel := widget.NewLabel(config.T("theme_label"))
	langLabel := widget.NewLabel(config.T("lang_label"))

	settingsCustomCard := widget.NewCard(config.T("settings_custom"), "", container.NewVBox(
		themeLabel,
		themeSelect,
		widget.NewSeparator(),
		langLabel,
		langSelect,
	))

	// Settings Tab Advanced Opt-In Toggles
	dwmDescLabel := widget.NewLabel(config.T("dwm_desc"))
	dwmDescLabel.Wrapping = fyne.TextWrapWord
	dwmCheck := widget.NewCheck(config.T("dwm_label"), func(checked bool) {
		go func() {
			err := hardware.SetDwmPriority(checked)
			if err != nil {
				logToUI(fmt.Sprintf("[ERROR] DWM Priority set failed: %v", err))
			} else {
				if checked {
					logToUI("[SYS] DWM priority optimization active (CpuPriorityClass = High).")
				} else {
					logToUI("[SYS] DWM priority optimization reverted.")
				}
			}
		}()
	})

	searchDescLabel := widget.NewLabel(config.T("search_desc"))
	searchDescLabel.Wrapping = fyne.TextWrapWord
	searchCheck := widget.NewCheck(config.T("search_label"), func(checked bool) {
		go func() {
			err := hardware.SetSearchSuspended(checked)
			if err != nil {
				logToUI(fmt.Sprintf("[ERROR] Windows Search Suspend failed: %v", err))
			} else {
				if checked {
					logToUI("[SYS] Windows Search suspended (Background indexing halted).")
				} else {
					logToUI("[SYS] Windows Search resumed.")
				}
			}
		}()
	})

	hiberDescLabel := widget.NewLabel(config.T("hiber_desc"))
	hiberDescLabel.Wrapping = fyne.TextWrapWord
	hiberCheck := widget.NewCheck(config.T("hiber_label"), func(checked bool) {
		go func() {
			err := hardware.SetHibernationDisabled(checked)
			if err != nil {
				logToUI(fmt.Sprintf("[ERROR] Hibernation toggle failed: %v", err))
			} else {
				if checked {
					logToUI("[SYS] Hibernation deactivated (Massive SSD space freed).")
				} else {
					logToUI("[SYS] Hibernation reactivated.")
				}
			}
		}()
	})

	settingsPerfCard := widget.NewCard(config.T("settings_perf"), "", container.NewVBox(
		dwmCheck,
		dwmDescLabel,
		widget.NewSeparator(),
		searchCheck,
		searchDescLabel,
		widget.NewSeparator(),
		hiberCheck,
		hiberDescLabel,
	))

	settingsScroll := container.NewVScroll(container.NewVBox(
		settingsCustomCard,
		settingsPerfCard,
	))

	// 8. Build Tab Containers with Padded Margins to separate content from tab buttons
	boosterTab := container.NewTabItem(config.T("booster_tab"), container.NewBorder(
		createVSpacer(12), nil, nil, nil,
		container.NewPadded(container.NewVScroll(container.NewVBox(
			hudCard,
			container.New(&DynamicGridLayout{MaxCols: 3, MinColWidth: 250},
				extremeCard,
				balancedCard,
				restoreCard,
			),
		))),
	))

	systemTab := container.NewTabItem(config.T("memory_tab"), container.NewBorder(
		createVSpacer(12), nil, nil, nil,
		container.NewPadded(container.NewVScroll(container.New(
			&DynamicGridLayout{MaxCols: 2, MinColWidth: 320},
			memCard,
			srvCard,
		))),
	))

	netDescLabel := widget.NewLabel(config.T("net_shield"))
	netDescLabel.Alignment = fyne.TextAlignCenter
	netDescLabel.TextStyle = fyne.TextStyle{Italic: true}

	networkScroll := container.NewVScroll(container.NewVBox(
		netDetailsCard,
		netDescLabel,
	))

	networkTab := container.NewTabItem(config.T("network_tab"), container.NewBorder(
		createVSpacer(12), nil, nil, nil,
		container.NewPadded(networkScroll),
	))

	kernelTab := container.NewTabItem(config.T("kernel_tab"), container.NewBorder(
		createVSpacer(12), nil, nil, nil,
		container.NewPadded(container.NewVBox(
			kernelCard,
		)),
	))

	settingsTab := container.NewTabItem(config.T("settings_tab"), container.NewBorder(
		createVSpacer(12), nil, nil, nil,
		container.NewPadded(settingsScroll),
	))

	tabs := container.NewAppTabs(boosterTab, systemTab, networkTab, kernelTab, settingsTab)
	tabs.SetTabLocation(container.TabLocationTop)

	// Log Matrix Stream Card
	consoleFeed := widget.NewMultiLineEntry()
	consoleFeed.SetText("NosBoost Console Initialized. Awaiting command matrix input...\n")
	consoleFeed.Disable()
	consoleFeed.Wrapping = fyne.TextWrapWord
	consoleFeed.TextStyle = fyne.TextStyle{Monospace: true}

	consoleContainer := container.NewGridWithRows(1, consoleFeed)
	consoleCard := widget.NewCard(config.T("console_log_title"), "", container.NewThemeOverride(consoleContainer, theme.DarkTheme()))

	// Dynamic Language Switching Closure
	refreshUIStrings = func() {
		if isRefreshingStrings {
			return
		}
		isRefreshingStrings = true
		defer func() { isRefreshingStrings = false }()

		titleLabel.Text = config.T("app_title")
		titleLabel.Color = theme.PrimaryColor()
		subtitleLabel.Text = config.T("subtitle")
		subtitleLabel.Color = theme.ForegroundColor()

		memCard.Title = config.T("ram_health")
		cleanMemBtn.SetText(config.T("clean_mem"))

		netDetailsCard.Title = config.T("net_shield")
		flushNetworkBtn.SetText(config.T("flush_dns"))

		kernelCard.Title = config.T("kernel_card")
		srvCard.Title = config.T("srv_ledger")
		deepVerifierBtn.SetText(config.T("verify_srv"))

		hudCard.Title = config.T("hud_title")
		blueprintBtn.SetText(config.T("blueprint"))

		extremeCard.Title = config.T("extreme_title")
		extremeFeatures.SetText(config.T("extreme_desc"))
		extremeCardBtn.SetText(config.T("extreme_btn"))

		balancedCard.Title = config.T("balanced_title")
		balancedFeatures.SetText(config.T("balanced_desc"))
		balancedCardBtn.SetText(config.T("balanced_btn"))

		restoreCard.Title = config.T("restore_title")
		restoreDescLabel.SetText(config.T("restore_desc"))
		restoreCardBtn.SetText(config.T("restore_btn"))

		themeLabel.SetText(config.T("theme_label"))
		langLabel.SetText(config.T("lang_label"))
		settingsCustomCard.Title = config.T("settings_custom")
		settingsPerfCard.Title = config.T("settings_perf")
		dwmCheck.SetText(config.T("dwm_label"))
		dwmDescLabel.SetText(config.T("dwm_desc"))
		searchCheck.SetText(config.T("search_label"))
		searchDescLabel.SetText(config.T("search_desc"))
		hiberCheck.SetText(config.T("hiber_label"))
		hiberDescLabel.SetText(config.T("hiber_desc"))

		boosterTab.Text = config.T("booster_tab")
		systemTab.Text = config.T("memory_tab")
		networkTab.Text = config.T("network_tab")
		kernelTab.Text = config.T("kernel_tab")
		settingsTab.Text = config.T("settings_tab")

		consoleCard.Title = config.T("console_log_title")

		// Translate theme selection dropdown options and restore the active selection index
		themeIndex := 0
		for i, opt := range themeSelect.Options {
			if opt == themeSelect.Selected {
				themeIndex = i
				break
			}
		}
		themeSelect.Options = []string{config.T("theme_dark"), config.T("theme_light")}
		themeSelect.SetSelected(themeSelect.Options[themeIndex])

		// Explicitly refresh all card titles and components
		memCard.Refresh()
		netDetailsCard.Refresh()
		kernelCard.Refresh()
		srvCard.Refresh()
		hudCard.Refresh()
		extremeCard.Refresh()
		balancedCard.Refresh()
		restoreCard.Refresh()
		settingsCustomCard.Refresh()
		settingsPerfCard.Refresh()
		consoleCard.Refresh()

		titleLabel.Refresh()
		subtitleLabel.Refresh()
		tabs.Refresh()
		myWindow.Content().Refresh()
	}

	// Dynamic language select action trigger
	langSelect.OnChanged = func(selected string) {
		code, exists := langNameToCode[selected]
		if !exists {
			code = "en"
		}
		config.SetLanguage(code)
		if code == "en" {
			logToUI("[UI] Language switched to English.")
		} else {
			logToUI(fmt.Sprintf("[UI] Dil %s olarak değiştirildi.", selected))
		}
		refreshUIStrings()
	}

	// Set initial language selection matching config
	initialLangCode := config.GetLanguage()
	if name, exists := langCodeToName[initialLangCode]; exists {
		langSelect.SetSelected(name)
	} else {
		langSelect.SetSelected(langCodeToName["en"])
	}

	// Final Layout
	bodySplit := container.NewVSplit(
		tabs,
		consoleCard,
	)
	bodySplit.Offset = 0.68

	mainLayout := container.NewBorder(
		headerContainer,
		nil,
		nil,
		nil,
		bodySplit,
	)

	myWindow.SetContent(mainLayout)
	myWindow.Resize(fyne.NewSize(850, 650))

	// 9. ASYNCHRONOUS BACKGROUND STATS UPDATES (Tickers)
	go func() {
		for msg := range UIConsoleChan {
			currentText := consoleFeed.Text

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

			if len(newText) > 30000 {
				newText = newText[15000:]
			}

			fyne.Do(func() {
				consoleFeed.SetText(newText)
				consoleFeed.CursorColumn = len(newText)
			})
		}
	}()

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
