package network

import (
	"fmt"
	"os/exec"
)

// FlushNetworkStack executes atomic optimization sequences to sanitize the Windows network stack.
// It flushes the DNS cache, resets Winsock catalog indexes, and recycles IPv4 interfaces.
func FlushNetworkStack() error {
	// 1. Flush DNS cache
	dnsCmd := exec.Command("ipconfig", "/flushdns")
	if err := dnsCmd.Run(); err != nil {
		return fmt.Errorf("failed to flush DNS cache: %w", err)
	}

	// 2. Reset Winsock catalog
	winsockCmd := exec.Command("netsh", "winsock", "reset")
	if err := winsockCmd.Run(); err != nil {
		return fmt.Errorf("failed to reset Winsock catalog: %w", err)
	}

	// 3. Reset TCP/IP stack configuration
	ipCmd := exec.Command("netsh", "int", "ip", "reset")
	if err := ipCmd.Run(); err != nil {
		return fmt.Errorf("failed to recycle TCP/IP interface stack: %w", err)
	}

	return nil
}
