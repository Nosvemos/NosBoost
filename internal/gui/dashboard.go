package gui

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.org/x/sys/windows"
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatus = kernel32.NewProc("GlobalMemoryStatusEx")
)

// ShowDashboard bootstraps and renders the Fyne native graphical interface.
func ShowDashboard() {
	// 1. Force strict dark-theme override at environment level
	os.Setenv("FYNE_THEME", "dark")

	myApp := app.New()
	myWindow := myApp.NewWindow("NosBoost // Active Performance Matrix")
	myWindow.Resize(fyne.NewSize(850, 600))
	myWindow.SetFixedSize(true)

	// 2. Setup Neon-Green and Dark Theme Accent Elements
	titleLabel := canvas.NewText("🚀 NOSBOOST OPTIMIZATION SUITE", theme.PrimaryColor())
	titleLabel.TextSize = 20
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	subtitleLabel := canvas.NewText("Windows Kernel Systems Latency Enforcement Engine", theme.ForegroundColor())
	subtitleLabel.TextSize = 11
	subtitleLabel.Alignment = fyne.TextAlignCenter

	headerContainer := container.NewVBox(
		titleLabel,
		subtitleLabel,
		canvas.NewLine(theme.PrimaryColor()),
	)

	// 3. TELEMETRY MONITOR WIDGETS
	// RAM Health Metrics
	ramLabel := widget.NewLabel("Memory Allocation: Querying...")
	ramLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Server Region Ping Indicators
	pingUSALabel := widget.NewLabel("NA East: Measuring...")
	pingUSALabel.TextStyle = fyne.TextStyle{Monospace: true}
	pingUSAWestLabel := widget.NewLabel("NA West: Measuring...")
	pingUSAWestLabel.TextStyle = fyne.TextStyle{Monospace: true}
	pingEULabel := widget.NewLabel("Europe: Measuring...")
	pingEULabel.TextStyle = fyne.TextStyle{Monospace: true}
	pingAsiaLabel := widget.NewLabel("Asia Pacific: Measuring...")
	pingAsiaLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// System Architecture locks
	systemStatusLabel := widget.NewLabel(fmt.Sprintf("Logical Processors: %d Cores", runtime.NumCPU()))
	systemStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	powerSchemeLabel := widget.NewLabel("Power Lock: Safe Baseline")
	powerSchemeLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Assemble Telemetry Boxes
	telemetryCard := widget.NewCard("📊 TELEMETRY ENGINE MONITOR", "", container.NewVBox(
		widget.NewLabel("Physical Memory State:"),
		ramLabel,
		canvas.NewLine(theme.DisabledColor()),
		widget.NewLabel("Regional Servers Ping (AWS TCP Gateways):"),
		container.NewGridWithColumns(2,
			pingUSALabel, pingUSAWestLabel,
			pingEULabel, pingAsiaLabel,
		),
		canvas.NewLine(theme.DisabledColor()),
		widget.NewLabel("Kernel Processor Locking:"),
		systemStatusLabel,
		powerSchemeLabel,
	))

	// 4. PROFILE SELECTOR CONTROLS
	extremeBtn := widget.NewButtonWithIcon("🔥 EXTREME COMPETITIVE", theme.ConfirmIcon(), func() {
		logToUI("[UI] User initiated EXTREME mode trigger.")
		go ApplyExtremeMode()
	})
	extremeBtn.Importance = widget.HighImportance

	balancedBtn := widget.NewButtonWithIcon("⚡ BALANCED GAMING", theme.MediaPlayIcon(), func() {
		logToUI("[UI] User initiated BALANCED mode trigger.")
		go ApplyBalancedMode()
	})

	restoreBtn := widget.NewButtonWithIcon("🛡️ TOTAL RESTORE", theme.CancelIcon(), func() {
		logToUI("[UI] User initiated TOTAL RESTORE trigger.")
		go ApplyTotalRestore()
	})
	restoreBtn.Importance = widget.DangerImportance

	// Power State Indicators
	activeModeLabel := widget.NewLabel("ACTIVE MATRIX: SAFE DEFAULT")
	activeModeLabel.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
	activeModeLabel.Alignment = fyne.TextAlignCenter

	controlsCard := widget.NewCard("⚙️ ONE-CLICK ORCHESTRATION", "", container.NewVBox(
		widget.NewLabel("Select desired latency profile vector:"),
		extremeBtn,
		widget.NewLabel(""),
		balancedBtn,
		widget.NewLabel(""),
		restoreBtn,
		canvas.NewLine(theme.DisabledColor()),
		activeModeLabel,
	))

	// 5. LIGHTWEIGHT CONSOLE LOG TRACKER
	consoleFeed := widget.NewMultiLineEntry()
	consoleFeed.SetText("NosBoost Console Initialized. Awaiting command matrix input...\n")
	consoleFeed.Disable()
	consoleFeed.Wrapping = fyne.TextWrapWord
	consoleFeed.TextStyle = fyne.TextStyle{Monospace: true}


	consoleContainer := container.NewGridWithRows(1, consoleFeed)
	consoleCard := widget.NewCard("🖥️ LOG MATRIX STATUS STREAM", "", consoleContainer)

	// 6. ASYNCHRONOUS BACKGROUND STATS UPDATES (Tickers)
	// Start thread to consume log channel safely without blocking UI
	go func() {
		for msg := range UIConsoleChan {
			currentText := consoleFeed.Text
			newText := currentText + msg + "\n"
			
			// Cap log size at 30k characters to prevent memory leaks
			if len(newText) > 30000 {
				newText = newText[15000:]
			}
			
			consoleFeed.SetText(newText)
			// Move cursor to bottom to auto-scroll
			consoleFeed.CursorColumn = len(newText)
		}
	}()

	// Spawn Telemetry Tickers in isolated background goroutines
	go runTelemetryTickers(ramLabel, powerSchemeLabel, activeModeLabel, pingUSALabel, pingUSAWestLabel, pingEULabel, pingAsiaLabel)

	// Assemble responsive grid container
	bodyContainer := container.NewGridWithColumns(2,
		telemetryCard,
		controlsCard,
	)

	// Main Grid Layout
	mainLayout := container.New(layout.NewBorderLayout(headerContainer, nil, nil, nil),
		headerContainer,
		container.NewGridWithRows(2,
			bodyContainer,
			consoleCard,
		),
	)

	myWindow.SetContent(mainLayout)
	logToUI("🛡️  NosBoost UI canvas initialized. Baseline state secure.")
	myWindow.ShowAndRun()
}

// runTelemetryTickers sweeps memory status and AWS gateways asynchronously
func runTelemetryTickers(
	ramLabel *widget.Label,
	powerLabel *widget.Label,
	activeModeLabel *widget.Label,
	pingUSA, pingUSAWest, pingEU, pingAsia *widget.Label,
) {
	memTicker := time.NewTicker(1500 * time.Millisecond)
	pingTicker := time.NewTicker(8 * time.Second)
	defer memTicker.Stop()
	defer pingTicker.Stop()

	// Direct gateways for latency estimation
	usEastGateway := "dynamodb.us-east-1.amazonaws.com:443"
	usWestGateway := "dynamodb.us-west-2.amazonaws.com:443"
	euGateway := "dynamodb.eu-central-1.amazonaws.com:443"
	asiaGateway := "dynamodb.ap-northeast-1.amazonaws.com:443"

	// Immediate run once
	updateMemoryTelemetry(ramLabel)
	updatePingTelemetry(pingUSA, usEastGateway, "NA East")
	updatePingTelemetry(pingUSAWest, usWestGateway, "NA West")
	updatePingTelemetry(pingEU, euGateway, "Europe")
	updatePingTelemetry(pingAsia, asiaGateway, "Asia Pacific")

	for {
		select {
		case <-memTicker.C:
			updateMemoryTelemetry(ramLabel)
			// Re-query power state
			mode := GetCurrentMode()
			activeModeLabel.SetText(fmt.Sprintf("ACTIVE MATRIX: %s", strings.ToUpper(mode)))
			if mode == "Safe Default" {
				powerLabel.SetText("Power Lock: Safe Baseline Plan")
			} else {
				powerLabel.SetText("Power Lock: Ultimate Performance (Active)")
			}
		case <-pingTicker.C:
			// Run pings concurrently to prevent thread block during TCP Handshake delay
			go updatePingTelemetry(pingUSA, usEastGateway, "NA East")
			go updatePingTelemetry(pingUSAWest, usWestGateway, "NA West")
			go updatePingTelemetry(pingEU, euGateway, "Europe")
			go updatePingTelemetry(pingAsia, asiaGateway, "Asia Pacific")
		}
	}
}

// updateMemoryTelemetry retrieves Windows Global Memory parameters
func updateMemoryTelemetry(label *widget.Label) {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))

	r1, _, err := procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&mem)))
	if r1 == 0 {
		label.SetText(fmt.Sprintf("Memory Allocation: Failed to query Windows APIs (%v)", err))
		return
	}

	totalGB := float64(mem.TotalPhys) / (1024 * 1024 * 1024)
	availGB := float64(mem.AvailPhys) / (1024 * 1024 * 1024)
	allocGB := totalGB - availGB
	percent := mem.MemoryLoad

	label.SetText(fmt.Sprintf("Allocated: %.2f GB / %.2f GB (%d%%)", allocGB, totalGB, percent))
}

// updatePingTelemetry measures fast TCP gateway Handshake delays
func updatePingTelemetry(label *widget.Label, target, regionName string) {
	start := time.Now()
	// Short timeout prevents long UI lockups during network dropout
	conn, err := net.DialTimeout("tcp", target, 1200*time.Millisecond)
	if err != nil {
		label.SetText(fmt.Sprintf("%s: Offline", regionName))
		return
	}
	conn.Close()
	ms := time.Since(start).Milliseconds()
	label.SetText(fmt.Sprintf("%s: %d ms", regionName, ms))
}
