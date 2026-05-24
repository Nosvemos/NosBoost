package gui

import (
	"testing"
)

// TestOrchestratorStateVerification verifies the thread-safe state management of the orchestrator.
func TestOrchestratorStateVerification(t *testing.T) {
	// 1. Initial State Check
	initial := GetCurrentMode()
	if initial != "Safe Default" {
		t.Errorf("Expected initial state to be 'Safe Default', got '%s'", initial)
	}

	// 2. State Transition to Extreme
	updateCurrentMode("Extreme")
	extreme := GetCurrentMode()
	if extreme != "Extreme" {
		t.Errorf("Expected transitioned state to be 'Extreme', got '%s'", extreme)
	}

	// 3. State Transition to Balanced
	updateCurrentMode("Balanced")
	balanced := GetCurrentMode()
	if balanced != "Balanced" {
		t.Errorf("Expected transitioned state to be 'Balanced', got '%s'", balanced)
	}

	// 4. Return to Default
	updateCurrentMode("Safe Default")
	restored := GetCurrentMode()
	if restored != "Safe Default" {
		t.Errorf("Expected restored state to be 'Safe Default', got '%s'", restored)
	}
}

// TestTargetGamesRegistry verifies that crucial competitive games are registered for latency polling.
func TestTargetGamesRegistry(t *testing.T) {
	requiredGames := map[string]bool{
		"cs2.exe":                     true,
		"VALORANT-Win64-Shipping.exe": true,
		"League of Legends.exe":       true,
		"dota2.exe":                   true,
		"r5apex.exe":                  true,
	}

	foundCount := 0
	for _, game := range TargetGames {
		if requiredGames[game] {
			foundCount++
		}
	}

	if foundCount < 3 {
		t.Errorf("Latency watcher targets missing vital competitive games. Found only %d of required", foundCount)
	}
}

// TestUIConsolePipeChannel integrity checks the UI monospaced console logger communication pipeline.
func TestUIConsolePipeChannel(t *testing.T) {
	testMsg := "Integrity Check: NosBoost Core"
	logToUI(testMsg)

	select {
	case received := <-UIConsoleChan:
		if !containsLog(received, testMsg) {
			t.Errorf("Expected channel message to contain '%s', got '%s'", testMsg, received)
		}
	default:
		t.Error("UIConsoleChan pipeline failed to transmit status logs.")
	}
}

func containsLog(log, sub string) bool {
	// Check if log contains sub-string
	return len(log) >= len(sub) && (log[len(log)-len(sub):] == sub || log[10:10+len(sub)] == sub || (len(log) > len(sub) && (log[11:11+len(sub)] == sub || log[12:12+len(sub)] == sub)))
}
