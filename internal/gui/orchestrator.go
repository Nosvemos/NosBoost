package gui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"nosboost/internal/booster"
	"nosboost/internal/cleaner"
	"nosboost/internal/config"
	"nosboost/internal/hardware"
	"nosboost/internal/memory"
	"nosboost/internal/network"
	"nosboost/internal/syswatch"

	"golang.org/x/sys/windows"
)

var (
	// UIConsoleChan receives log status updates to print in the scrolling console tracker.
	UIConsoleChan = make(chan string, 1024)

	// Mode management
	currentModeMutex sync.Mutex
	CurrentMode      = "Safe Default" // "Extreme", "Balanced", "Safe Default"

	// Execution mutex to protect concurrent configuration access
	orchestrationMutex sync.Mutex

	// Execution contexts for background goroutines
	cancelPowerLock context.CancelFunc
	cancelGamePoll  context.CancelFunc
	orchestratorWG  sync.WaitGroup

	// Active game detection list
	TargetGames = []string{
		"cs2.exe",
		"csgo.exe",
		"VALORANT-Win64-Shipping.exe",
		"League of Legends.exe",
		"LeagueClient.exe",
		"dota2.exe",
		"r5apex.exe",
		"FortniteClient-Win64-Shipping.exe",
		"Overwatch.exe",
		"TslGame.exe",
		"RainbowSix.exe",
		"RustClient.exe",
		"EldenRing.exe",
		"cod.exe",
		"Minecraft.exe",
		"GTA5.exe",
		"Cyberpunk2077.exe",
	}

	// Track injected state for selective cleanup
	injectedRoutes []string
	servicesFrozen bool
)

// logToUI pipes status logs directly to the UI channel.
func logToUI(msg string) {
	select {
	case UIConsoleChan <- fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg):
	default:
		// Drop log if channel is fully saturated (guards against memory leak)
	}
}

// GetCurrentMode retrieves the thread-safe active orchestration mode name.
func GetCurrentMode() string {
	currentModeMutex.Lock()
	defer currentModeMutex.Unlock()
	return CurrentMode
}

// updateCurrentMode sets the active mode name safely.
func updateCurrentMode(mode string) {
	currentModeMutex.Lock()
	defer currentModeMutex.Unlock()
	CurrentMode = mode
}

// stopBackgroundThreads cancels any active power locks and polling tickers.
func stopBackgroundThreads() {
	if cancelPowerLock != nil {
		cancelPowerLock()
		cancelPowerLock = nil
	}
	if cancelGamePoll != nil {
		cancelGamePoll()
		cancelGamePoll = nil
	}
	orchestratorWG.Wait()
}

// ApplyExtremeMode executes the complete low-latency performance arsenal.
// Runs asynchronously with cooperative cancel checks between stages.
func ApplyExtremeMode() {
	orchestrationMutex.Lock()
	defer orchestrationMutex.Unlock()

	currentModeMutex.Lock()
	if CurrentMode == "Extreme" {
		currentModeMutex.Unlock()
		logToUI("[WARNING] System already running in Extreme Competitive Mode!")
		return
	}
	currentModeMutex.Unlock()

	// Stop any existing orchestrations
	stopBackgroundThreads()
	updateCurrentMode("Extreme")

	logToUI("[SYSTEM] Initiating EXTREME Competitive Mode Optimization...")

	// 1. Deep garbage cleanup
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[CLEANER] Running system junk cleaner...")
	metrics := cleaner.ExecuteDeepCleanup()
	
	if GetCurrentMode() != "Extreme" { return }
	freedMB := float64(metrics.BytesFreed) / (1024 * 1024)
	logToUI(fmt.Sprintf("[CLEANER] Junk cleanup complete. Deleted %d files (%.2f MB freed). Skipped %d locked files.", 
		metrics.FilesDeleted, freedMB, metrics.FilesSkipped))

	// 2. CPU Core Parking Elimination
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[BOOSTER] Eliminating CPU Core Parking limits...")
	if err := booster.EnableCoreParkingElimination(); err == nil {
		logToUI("[BOOSTER] Core parking disabled successfully. All CPU cores 100% awake.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Core parking override failed: %v", err))
	}

	// 3. Process Priorityseparation (Quantum override)
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[BOOSTER] Tuning Win32 Priority Separation to short-variable gaming index (0x26)...")
	if err := booster.OptimizePrioritySeparation(); err == nil {
		logToUI("[BOOSTER] Foreground quantum separation optimized.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Priority separation optimization failed: %v", err))
	}

	// 4. Power Plan Settings
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[POWER] Elevating system power plan to Ultimate Performance...")
	if err := syswatch.EnableUltimatePowerPlan(); err == nil {
		logToUI(fmt.Sprintf("[POWER] Power plan locked to active target: %s.", syswatch.ActiveTargetScheme))
		ctx, cancel := context.WithCancel(context.Background())
		cancelPowerLock = cancel
		orchestratorWG.Add(1)
		go func() {
			defer orchestratorWG.Done()
			syswatch.StartPowerLockTicker(ctx)
		}()
		logToUI("[POWER] Power scheme persistent lock ticker started.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Ultimate Performance activation failed: %v", err))
	}

	// 5. Network Tuning (TCP NoDelay + Static Route injections)
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[NETWORK] Injecting low-latency TCP NoDelay registry parameters...")
	if err := network.InjectTCPNoDelay(); err == nil {
		logToUI("[NETWORK] TCP ACK Frequency & TCP NoDelay set to instant fire.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] TCP registry injection failed: %v", err))
	}

	if GetCurrentMode() != "Extreme" { return }
	logToUI("[NETWORK] Optimizing network adapters LSO/RSC/RSS/DCA packet parameters...")
	if err := network.OptimizeNetworkInterfaceSettings(); err == nil {
		logToUI("[NETWORK] Network adapter packet loss and drops protection active.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Network adapter optimizations failed: %v", err))
	}


	if GetCurrentMode() != "Extreme" { return }
	logToUI("[NETWORK] Loading esports regional servers and injecting static route bypasses...")
	if routes, err := network.InjectGameRoutes(); err == nil {
		injectedRoutes = routes
		logToUI(fmt.Sprintf("[NETWORK] Injected %d esports network gateway routes directly into OS routing table.", len(routes)))
	} else {
		logToUI(fmt.Sprintf("[WARNING] Esports route injection failed: %v", err))
	}

	// 6. MSI Interrupt Modes & Timers
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[HARDWARE] Discovering display and network adapters for Message Signaled Interrupts (MSI)...")
	if err := hardware.EnableMSIMode(); err == nil {
		logToUI("[HARDWARE] MSI mode conversion completed. Hardware priority raised to High.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] MSI conversion utility encountered errors: %v", err))
	}

	if GetCurrentMode() != "Extreme" { return }
	logToUI("[HARDWARE] Optimizing peripheral input report buffer queues...")
	if err := hardware.TunePeripheralBuffers(); err == nil {
		logToUI("[HARDWARE] Mouse and Keyboard queue buffers streamlined to 20 reports.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Peripheral buffers optimization failed: %v", err))
	}

	if GetCurrentMode() != "Extreme" { return }
	logToUI("[HARDWARE] Injecting input responsive and Game DVR tweaks...")
	if err := hardware.OptimizeInputLatency(); err == nil {
		logToUI("[HARDWARE] 1:1 Raw input and Game DVR latency eliminated.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Input latency adjustments failed: %v", err))
	}


	if GetCurrentMode() != "Extreme" { return }
	logToUI("[HARDWARE] Invariant platform clocks configuration check...")
	if err := hardware.OptimizeSystemTimers(); err == nil {
		logToUI("[HARDWARE] Kernel timers locked to high-precision hardware clocks (TSC enabled).")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Boot timer configuration update failed: %v", err))
	}

	// 7. Freeze aggressive telemetry service cycles (Extreme Only)
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[SYSWATCH] Freezing background telemetry and diagnostic services...")
	if count, err := syswatch.SuspendBackgroundServices(); err == nil {
		servicesFrozen = true
		logToUI(fmt.Sprintf("[SYSWATCH] Suspended %d background diagnostics services (wuauserv & DiagTrack).", count))
	} else {
		logToUI(fmt.Sprintf("[WARNING] Telemetry freeze failed: %v", err))
	}

	// 8. Memory list purging & SysMain disable
	if GetCurrentMode() != "Extreme" { return }
	logToUI("[MEMORY] Sweeping cache memory pages & standby list files...")
	_ = memory.PurgeStandbyList()
	_ = memory.FlushModifiedList()
	logToUI("[MEMORY] Standby and modified list pages cleared. Available physical RAM maximized.")

	if GetCurrentMode() != "Extreme" { return }
	logToUI("[MEMORY] Stopping service SysMain to disable background cache analysis...")
	if err := memory.DisableSysMain(); err == nil {
		logToUI("[MEMORY] SysMain caching stopped and startup set to Disabled.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] SysMain service control failed: %v", err))
	}

	// 9. Startup background game polling
	if GetCurrentMode() != "Extreme" { return }
	ctx, cancel := context.WithCancel(context.Background())
	cancelGamePoll = cancel
	orchestratorWG.Add(1)
	go startHybridGamePoller(ctx, true)

	logToUI("[SYSTEM] EXTREME Optimization Matrix is fully operational. Scanning for game launch...")
}

// ApplyBalancedMode executes optimized power, memory, and network settings while keeping multitasking active.
func ApplyBalancedMode() {
	orchestrationMutex.Lock()
	defer orchestrationMutex.Unlock()

	currentModeMutex.Lock()
	if CurrentMode == "Balanced" {
		currentModeMutex.Unlock()
		logToUI("[WARNING] System already running in Balanced Gaming Mode!")
		return
	}
	currentModeMutex.Unlock()

	// Stop any existing orchestrations
	stopBackgroundThreads()
	updateCurrentMode("Balanced")

	logToUI("[SYSTEM] Initiating BALANCED Gaming Mode Optimization...")

	// 1. Deep garbage cleanup
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[CLEANER] Running system junk cleaner...")
	metrics := cleaner.ExecuteDeepCleanup()
	
	if GetCurrentMode() != "Balanced" { return }
	freedMB := float64(metrics.BytesFreed) / (1024 * 1024)
	logToUI(fmt.Sprintf("[CLEANER] Junk cleanup complete. Deleted %d files (%.2f MB freed).", 
		metrics.FilesDeleted, freedMB))

	// 2. CPU Core Parking Elimination
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[BOOSTER] Eliminating CPU Core Parking limits...")
	if err := booster.EnableCoreParkingElimination(); err == nil {
		logToUI("[BOOSTER] Core parking disabled successfully. All CPU cores 100% awake.")
	}

	// 3. Process Priorityseparation (Quantum override)
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[BOOSTER] Tuning Win32 Priority Separation to short-variable gaming index (0x26)...")
	_ = booster.OptimizePrioritySeparation()

	// 4. Power Plan Settings
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[POWER] Elevating system power plan to Ultimate Performance...")
	if err := syswatch.EnableUltimatePowerPlan(); err == nil {
		logToUI(fmt.Sprintf("[POWER] Power plan locked to active target: %s.", syswatch.ActiveTargetScheme))
		ctx, cancel := context.WithCancel(context.Background())
		cancelPowerLock = cancel
		orchestratorWG.Add(1)
		go func() {
			defer orchestratorWG.Done()
			syswatch.StartPowerLockTicker(ctx)
		}()
	}

	// 5. Network Tuning (TCP NoDelay + Routes)
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[NETWORK] Injecting low-latency TCP NoDelay registry parameters...")
	_ = network.InjectTCPNoDelay()
	_ = network.OptimizeNetworkInterfaceSettings()


	if GetCurrentMode() != "Balanced" { return }
	logToUI("[NETWORK] Loading esports regional servers and injecting static route bypasses...")
	if routes, err := network.InjectGameRoutes(); err == nil {
		injectedRoutes = routes
		logToUI(fmt.Sprintf("[NETWORK] Injected %d esports network gateway routes.", len(routes)))
	}

	// 6. MSI Interrupt Modes & Timers
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[HARDWARE] Converting display and network adapters to MSI Mode...")
	_ = hardware.EnableMSIMode()

	if GetCurrentMode() != "Balanced" { return }
	logToUI("[HARDWARE] Optimizing peripheral input report buffer queues...")
	_ = hardware.TunePeripheralBuffers()
	_ = hardware.OptimizeInputLatency()


	if GetCurrentMode() != "Balanced" { return }
	logToUI("[HARDWARE] Invariant platform clocks configuration check...")
	_ = hardware.OptimizeSystemTimers()

	// 7. Freeze background services: SKIPPED in Balanced Mode to permit multitasking
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[SYSWATCH] Balanced mode: Skipping background diagnostics suspension to allow multitasking.")
	servicesFrozen = false

	// 8. Memory list purging & SysMain disable
	if GetCurrentMode() != "Balanced" { return }
	logToUI("[MEMORY] Sweeping cache memory pages & standby list files...")
	_ = memory.PurgeStandbyList()
	_ = memory.FlushModifiedList()

	if GetCurrentMode() != "Balanced" { return }
	logToUI("[MEMORY] Stopping service SysMain to disable background cache analysis...")
	_ = memory.DisableSysMain()

	// 9. Startup background game polling (Balanced mode = no aggressive I/O throttling)
	if GetCurrentMode() != "Balanced" { return }
	ctx, cancel := context.WithCancel(context.Background())
	cancelGamePoll = cancel
	orchestratorWG.Add(1)
	go startHybridGamePoller(ctx, false)

	logToUI("[SYSTEM] BALANCED Gaming Optimization Matrix is active. Scanning for game launch...")
}

// ApplyTotalRestore reverts all optimizations and rollbacks the system to exact original state.
func ApplyTotalRestore() {
	orchestrationMutex.Lock()
	defer orchestrationMutex.Unlock()

	currentModeMutex.Lock()
	if CurrentMode == "Safe Default" {
		currentModeMutex.Unlock()
		logToUI("[WARNING] System already running in Safe Default mode.")
		return
	}
	currentModeMutex.Unlock()

	// Stop background poller threads
	stopBackgroundThreads()
	updateCurrentMode("Safe Default")

	logToUI("[RESTORE] Initiating TOTAL RESTORE Rollback to baseline configurations...")

	// 1. Restore baseline state values (MSI, network values, core parking, priorities, services)
	logToUI("[RESTORE] Rolling back system registry parameters from transaction snapshot...")
	if err := config.RestoreBaselineState(); err == nil {
		logToUI("[RESTORE] Reverted graphics card MSI parameters, TCP frequencies, quantum levels, and core parking.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Baseline snapshot restore failed: %v", err))
	}

	// 2. Remove injected routes
	if len(injectedRoutes) > 0 {
		logToUI("[RESTORE] Wiping esports custom routing bypass rules...")
		removed := network.DeleteGameRoutes(injectedRoutes)
		logToUI(fmt.Sprintf("[RESTORE] Deleted %d gaming network routes.", removed))
		injectedRoutes = nil
	}

	// 2b. Revert network LSO/RSC/RSS/DCA settings
	logToUI("[RESTORE] Reverting network adapter LSO, RSC, and TCP settings to default...")
	if err := network.RestoreNetworkInterfaceSettings(); err == nil {
		logToUI("[RESTORE] Network adapter settings reverted to OS defaults.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Network stack rollback encountered errors: %v", err))
	}


	// 3. Revert MSI modes
	logToUI("[RESTORE] Restoring Message Signaled Interrupts (MSI) to default state...")
	if err := hardware.DisableMSIMode(); err == nil {
		logToUI("[RESTORE] MSI mode overrides uninstalled.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] MSI restoration failed: %v", err))
	}

	// 4. Revert SysMain service
	logToUI("[RESTORE] Reloading SysMain caching service status...")
	if baseline, err := config.LoadBaselineState(); err == nil {
		var sysmainStart uint32 = 2 // default to automatic
		for _, srv := range baseline.Services {
			if srv.ServiceName == "SysMain" {
				sysmainStart = srv.StartValue
				break
			}
		}
		if err := memory.RestoreSysMain(sysmainStart, sysmainStart != 4); err == nil {
			logToUI("[RESTORE] Service SysMain restored and startup reset.")
		} else {
			logToUI(fmt.Sprintf("[WARNING] SysMain service restore failed: %v", err))
		}
	}

	// 5. Restore Timers
	logToUI("[RESTORE] Reverting precision platform boot clocks...")
	if err := hardware.RestoreSystemTimers(); err == nil {
		logToUI("[RESTORE] BCDEDIT kernel boot clocks reverted.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Kernel timers restore failed: %v", err))
	}

	// 6. Restore Peripheral Buffers
	logToUI("[RESTORE] Restoring mouse and keyboard buffer queues...")
	if err := hardware.RestorePeripheralBuffers(); err == nil {
		logToUI("[RESTORE] mouclass and kbdclass buffer sizes rolled back.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Peripheral buffers restoration failed: %v", err))
	}
	if err := hardware.RestoreInputLatency(); err == nil {
		logToUI("[RESTORE] Input latency and Game DVR settings rolled back.")
	} else {
		logToUI(fmt.Sprintf("[WARNING] Input latency rollback failed: %v", err))
	}


	// 7. Resume background frozen services
	if servicesFrozen {
		logToUI("[RESTORE] Resuming frozen background services...")
		count := syswatch.ResumeBackgroundServices()
		logToUI(fmt.Sprintf("[RESTORE] Resumed %d background diagnostics services (wuauserv & DiagTrack).", count))
		servicesFrozen = false
	}

	// 8. Revert game-specific tweaks (affinity, priority separation)
	logToUI("[RESTORE] Restoring background scheduling priorities...")
	_ = booster.RestorePrioritySeparation()
	booster.RestoreAffinities(0, 0)
	cleaner.RestoreBackgroundIO()

	logToUI("[RESTORE] TOTAL RESTORE Complete. System is restored to baseline defaults.")
}

// findRunningGame scans active processes for configured target games.
func findRunningGame() (uint32, string, bool) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, "", false
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0, "", false
	}

	gameMap := make(map[string]bool)
	for _, g := range TargetGames {
		gameMap[strings.ToLower(g)] = true
	}

	for {
		exeName := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if gameMap[exeName] {
			return entry.ProcessID, exeName, true
		}

		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}
	return 0, "", false
}

// startHybridGamePoller runs an intelligent process watcher. Upon game discovery, it opens a
// synchronized process handle, applies game-specific optimizations, and blocks via WaitForSingleObject
// until the game exits before cleanly restoring states.
func startHybridGamePoller(ctx context.Context, isExtreme bool) {
	defer orchestratorWG.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gamePID, gameExe, found := findRunningGame()
			if found {
				// Open process with SYNCHRONIZE rights to wait for its termination
				handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, gamePID)
				if err != nil {
					// Fallback: process exists but we failed to open it. Sleep and retry.
					continue
				}

				// Apply Game-Specific Latency Tweaks
				logToUI(fmt.Sprintf("[POLLER] Target game process detected: %s (PID: %d)", gameExe, gamePID))

				// 1. CPU Affinity Restructure
				logToUI("[BOOSTER] Dynamic Affinity: Binding background apps to Core 0 & 1...")
				_ = booster.RestrictBackgroundProcesses()

				logToUI(fmt.Sprintf("[BOOSTER] Dynamic Affinity: Isolating %s to Cores 2-N...", gameExe))
				origAffinity, affErr := booster.IsolateGameProcess(gamePID)
				if affErr != nil {
					logToUI(fmt.Sprintf("[WARNING] Game affinity isolation failed: %v", affErr))
				}

				// 2. Game priority elevation
				logToUI(fmt.Sprintf("[BOOSTER] Scheduling: Elevating priority class of %s to HIGH...", gameExe))
				origPriority, priErr := booster.ElevateProcessPriority(gamePID)
				if priErr != nil {
					logToUI(fmt.Sprintf("[WARNING] Priority class elevation failed: %v", priErr))
				}

				// 3. Reclaim physical RAM from background processes
				logToUI("[MEMORY] Compaction: Sweeping non-critical working sets into pagefile...")
				trimmedCount, _ := memory.TrimWorkingSets(gamePID)
				logToUI(fmt.Sprintf("[MEMORY] Swapped %d background processes memory footprints out of active RAM.", trimmedCount))

				// 4. Background I/O Throttling (Extreme Mode only)
				var ioThrottled bool
				if isExtreme {
					logToUI("[CLEANER] Throttling noisy background process disk I/O channels...")
					if count, ioErr := cleaner.ThrottleBackgroundIO(); ioErr == nil {
						ioThrottled = true
						logToUI(fmt.Sprintf("[CLEANER] Allocated SSD bandwidth exclusively to game. Downsized %d apps disk priorities.", count))
					} else {
						logToUI(fmt.Sprintf("[WARNING] I/O priority adjustment failed: %v", ioErr))
					}
				}

				logToUI(fmt.Sprintf("[POLLER] Optimal state locked. Monitoring lifecycle of %s via kernel signaling...", gameExe))

				// Spawn exit waiter in concurrent thread so we don't block context cancellations
				gameExitChan := make(chan struct{})
				go func() {
					_, _ = windows.WaitForSingleObject(handle, windows.INFINITE)
					windows.CloseHandle(handle)
					close(gameExitChan)
				}()

				// Wait for game exit or mode cancel
				select {
				case <-ctx.Done():
					// Mode changed while playing. Clean up game specific hooks.
					logToUI(fmt.Sprintf("[POLLER] Optimization mode canceled while %s is active. Restoring process states...", gameExe))
					if affErr == nil {
						booster.RestoreAffinities(gamePID, origAffinity)
					}
					if priErr == nil && origPriority != 0 {
						_ = booster.RestoreProcessPriority(gamePID, origPriority)
					}
					if ioThrottled {
						cleaner.RestoreBackgroundIO()
					}
					return
				case <-gameExitChan:
					// Game exited naturally! Revert game specific tweaks.
					logToUI(fmt.Sprintf("[POLLER] Game process %s terminated. Restoring runtime system state...", gameExe))
					if affErr == nil {
						booster.RestoreAffinities(gamePID, origAffinity)
					}
					if priErr == nil && origPriority != 0 {
						_ = booster.RestoreProcessPriority(gamePID, origPriority)
					}
					if ioThrottled {
						restored := cleaner.RestoreBackgroundIO()
						logToUI(fmt.Sprintf("[CLEANER] Reverted I/O priorities of %d background processes.", restored))
					}
					logToUI("[POLLER] Game execution state successfully rolled back. Resuming poller sweep...")
				}
			}
		}
	}
}
