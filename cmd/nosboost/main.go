package main

import (
	"fmt"
	"os"

	"nosboost/internal/config"
	"nosboost/internal/gui"

	"golang.org/x/sys/windows"
)

// isAdmin checks if the current process is running with elevated administrative privileges.
func isAdmin() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("NosBoost Engine - Administrative Launcher")
	fmt.Println("==================================================")

	// Pre-populate UI console channel with bootstrap logs
	gui.UIConsoleChan <- "[SYSTEM] Bootstrapping NosBoost engine services..."

	if !isAdmin() {
		fmt.Println("WARNING: NosBoost is NOT running with Administrative privileges!")
		gui.UIConsoleChan <- "[CRITICAL WARNING] NosBoost was launched WITHOUT Administrative privileges!"
		gui.UIConsoleChan <- "[CRITICAL WARNING] Kernel adjustments (Registry, Services, MSI, network routing)"
		gui.UIConsoleChan <- "[CRITICAL WARNING] require elevated authority. Please relaunch NosBoost as Administrator."
	} else {
		fmt.Println("Elevated privileges detected. Running with full system authority.")
		gui.UIConsoleChan <- "[SYSTEM] Administrative authority verified."

		// Guard: Capture baseline state ONLY if it doesn't already exist.
		// This protects the original baseline from being overwritten by an optimized state.
		if _, err := os.Stat(config.BackupFileName); os.IsNotExist(err) {
			gui.UIConsoleChan <- "[SYSTEM] No baseline backup found. Capturing original OS configuration..."
			state, err := config.SaveBaselineState()
			if err != nil {
				gui.UIConsoleChan <- fmt.Sprintf("[ERROR] Failed to save baseline state: %v", err)
			} else {
				gui.UIConsoleChan <- fmt.Sprintf("[SYSTEM] Baseline snapshot saved to %s (captured: %s).", config.BackupFileName, state.Timestamp)
			}
		} else {
			gui.UIConsoleChan <- fmt.Sprintf("[SYSTEM] Active baseline backup detected (%s). Lock secured.", config.BackupFileName)
			// Dry-run load baseline to verify serialization integrity
			if _, err := config.LoadBaselineState(); err != nil {
				gui.UIConsoleChan <- fmt.Sprintf("[WARNING] Baseline file is corrupted: %v. Re-creation recommended.", err)
			} else {
				gui.UIConsoleChan <- "[SYSTEM] Transaction snapshot parsed successfully. Verification complete."
			}
		}
	}

	gui.UIConsoleChan <- "[SYSTEM] Launching native Fyne graphical dashboard..."
	
	// Start the main Fyne loop (this blocks until the window is closed)
	gui.ShowDashboard()

	// Graceful exit logs to standard output
	fmt.Println("NosBoost dashboard closed. Exiting cleanly.")
}
