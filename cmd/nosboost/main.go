package main

import (
	"fmt"

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
	fmt.Println("🚀 NosBoost Engine - Ultra-High-Performance Suite")
	fmt.Println("==================================================")

	if !isAdmin() {
		fmt.Println("⚠️  WARNING: NosBoost is NOT running with Administrative privileges!")
		fmt.Println("   Many low-level optimizations (Registry, Services, MSI, routing)")
		fmt.Println("   require administrator rights. Please restart as Administrator.")
		fmt.Println("==================================================")
	} else {
		fmt.Println("✅ Elevated privileges detected. Running with full system authority.")
		fmt.Println("==================================================")
	}

	fmt.Println("NosBoost initialized successfully. Awaiting command interface bootstrap...")
}
