package network

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"nosboost/internal/config"
	"nosboost/internal/system"

	"golang.org/x/sys/windows/registry"
)

const (
	TcpipInterfacesKey = `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
	SystemProfileKey   = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
)

// ActiveNICInfo holds the active online network interface details.
type ActiveNICInfo struct {
	GUID           string
	IPAddress      string
	DefaultGateway string
}

// GetActiveLocalIPs retrieves a list of all active online IPv4 addresses on the host.
func GetActiveLocalIPs() (map[string]bool, error) {
	ips := make(map[string]bool)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("failed to read interface addresses: %w", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips[ipnet.IP.String()] = true
			}
		}
	}
	return ips, nil
}

// DiscoverActiveNIC scans the Tcpip Interfaces registry tree, correlating IP addresses
// against online unicast IPs to return the active interface GUID and default gateway.
func DiscoverActiveNIC() (*ActiveNICInfo, error) {
	activeIPs, err := GetActiveLocalIPs()
	if err != nil {
		return nil, err
	}

	rootKey, err := registry.OpenKey(registry.LOCAL_MACHINE, TcpipInterfacesKey, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open Tcpip Interfaces registry: %w", err)
	}
	defer rootKey.Close()

	guids, err := rootKey.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read interface GUID keys: %w", err)
	}

	for _, guid := range guids {
		nicPath := TcpipInterfacesKey + `\` + guid
		nicKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		// 1. Resolve active IP address (can be REG_SZ or REG_MULTI_SZ)
		var ipStr string
		if ipVal, _, err := nicKey.GetStringValue("DhcpIPAddress"); err == nil && ipVal != "" {
			ipStr = ipVal
		} else if ips, _, err := nicKey.GetStringsValue("IPAddress"); err == nil && len(ips) > 0 && ips[0] != "0.0.0.0" {
			ipStr = ips[0]
		}

		// 2. Correlate registry IP to online active IPs
		if ipStr != "" && activeIPs[ipStr] {
			info := &ActiveNICInfo{
				GUID:      guid,
				IPAddress: ipStr,
			}

			// 3. Resolve active gateway IP
			if gwVal, _, err := nicKey.GetStringValue("DhcpDefaultGateway"); err == nil && gwVal != "" {
				info.DefaultGateway = gwVal
			} else if gws, _, err := nicKey.GetStringsValue("DefaultGateway"); err == nil && len(gws) > 0 && gws[0] != "" {
				info.DefaultGateway = gws[0]
			}

			// Fallback: If gateway is empty in the registry, parse from routing table
			if info.DefaultGateway == "" {
				if gwFallback, err := getDefaultGatewayFallback(); err == nil && gwFallback != "" {
					info.DefaultGateway = gwFallback
				}
			}

			nicKey.Close()
			return info, nil
		}

		nicKey.Close()
	}

	return nil, errors.New("active online network adapter registry entry not found")
}

// getDefaultGatewayFallback queries the system routing table for the default route to extract the gateway IP.
func getDefaultGatewayFallback() (string, error) {
	cmd := system.Command("route", "print", "0.0.0.0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			if fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
				gw := fields[2]
				if net.ParseIP(gw) != nil {
					return gw, nil
				}
			}
		}
	}
	return "", fmt.Errorf("default gateway route not found in routing table")
}

// InjectTCPNoDelay configures immediately-flushed network sockets and elevations for the active adapter.
func InjectTCPNoDelay() error {
	nic, err := DiscoverActiveNIC()
	if err != nil {
		return fmt.Errorf("failed to discover active adapter: %w", err)
	}

	// 1. Enforce TcpAckFrequency & TCPNoDelay under the active adapter key
	nicPath := TcpipInterfacesKey + `\` + nic.GUID
	nicKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open active adapter registry key: %w", err)
	}
	defer nicKey.Close()

	if err := nicKey.SetDWordValue("TcpAckFrequency", 1); err != nil {
		return fmt.Errorf("failed to set TcpAckFrequency: %w", err)
	}
	if err := nicKey.SetDWordValue("TCPNoDelay", 1); err != nil {
		return fmt.Errorf("failed to set TCPNoDelay: %w", err)
	}
	if err := nicKey.SetDWordValue("TcpDelAckTicks", 0); err != nil {
		return fmt.Errorf("failed to set TcpDelAckTicks: %w", err)
	}

	// 2. Enforce system responsiveness priority tweaks under HKLM Multimedia SystemProfile
	profileKey, err := registry.OpenKey(registry.LOCAL_MACHINE, SystemProfileKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open SystemProfile registry key: %w", err)
	}
	defer profileKey.Close()

	if err := profileKey.SetDWordValue("NetworkThrottlingIndex", 0xffffffff); err != nil {
		return fmt.Errorf("failed to disable NetworkThrottlingIndex: %w", err)
	}
	if err := profileKey.SetDWordValue("SystemResponsiveness", 0); err != nil {
		return fmt.Errorf("failed to set SystemResponsiveness priority: %w", err)
	}

	return nil
}

// RevertTCPNoDelay restores active NIC latency parameters from the recorded baseline state.
func RevertTCPNoDelay() error {
	nic, err := DiscoverActiveNIC()
	if err != nil {
		return fmt.Errorf("failed to discover active adapter: %w", err)
	}

	baseline, err := config.LoadBaselineState()
	if err != nil {
		return fmt.Errorf("failed to load baseline state: %w", err)
	}

	// 1. Revert active adapter TCP values
	nicPath := TcpipInterfacesKey + `\` + nic.GUID
	nicKey, err := registry.OpenKey(registry.LOCAL_MACHINE, nicPath, registry.SET_VALUE)
	if err == nil {
		// Look up original backup values for this GUID
		var foundOriginal bool
		for _, originalNic := range baseline.Network.NICs {
			if originalNic.InterfaceGUID == nic.GUID {
				foundOriginal = true
				if originalNic.TcpAckFrequencyExists {
					_ = nicKey.SetDWordValue("TcpAckFrequency", originalNic.TcpAckFrequencyValue)
				} else {
					_ = nicKey.DeleteValue("TcpAckFrequency")
				}
				if originalNic.TCPNoDelayExists {
					_ = nicKey.SetDWordValue("TCPNoDelay", originalNic.TCPNoDelayValue)
				} else {
					_ = nicKey.DeleteValue("TCPNoDelay")
				}
				if originalNic.TcpDelAckTicksExists {
					_ = nicKey.SetDWordValue("TcpDelAckTicks", originalNic.TcpDelAckTicksValue)
				} else {
					_ = nicKey.DeleteValue("TcpDelAckTicks")
				}
				break
			}
		}
		// If we couldn't find matching GUID backup, delete newly injected parameters
		if !foundOriginal {
			_ = nicKey.DeleteValue("TcpAckFrequency")
			_ = nicKey.DeleteValue("TCPNoDelay")
			_ = nicKey.DeleteValue("TcpDelAckTicks")
		}
		nicKey.Close()
	}

	// 2. Revert System Profile values
	profileKey, err := registry.OpenKey(registry.LOCAL_MACHINE, SystemProfileKey, registry.SET_VALUE)
	if err == nil {
		if baseline.Network.NetworkThrottlingExists {
			_ = profileKey.SetDWordValue("NetworkThrottlingIndex", baseline.Network.NetworkThrottlingValue)
		} else {
			_ = profileKey.DeleteValue("NetworkThrottlingIndex")
		}

		if baseline.Network.SystemResponsivenessExists {
			_ = profileKey.SetDWordValue("SystemResponsiveness", baseline.Network.SystemResponsivenessValue)
		} else {
			_ = profileKey.DeleteValue("SystemResponsiveness")
		}
		profileKey.Close()
	}

	return nil
}

// OptimizeNetworkInterfaceSettings disables LSO/RSC and configures global TCP settings (RSS, DCA, ECN) for maximum packet stability and low latency.
func OptimizeNetworkInterfaceSettings() error {
	// 1. Disable Large Send Offload (LSO) which causes high packet drops/loss
	_ = system.Exec("powershell", "-Command", "Disable-NetAdapterLso -Name * -IPv4 -Confirm:$false -ErrorAction SilentlyContinue")
	_ = system.Exec("powershell", "-Command", "Disable-NetAdapterLso -Name * -IPv6 -Confirm:$false -ErrorAction SilentlyContinue")

	// 2. Disable Receive Segment Coalescing (RSC) which causes packet latency/jitter stutters
	_ = system.Exec("powershell", "-Command", "Disable-NetAdapterRsc -Name * -Confirm:$false -ErrorAction SilentlyContinue")

	// 2b. Disable Packet Coalescing which delays packet interrupts to the CPU
	_ = system.Exec("powershell", "-Command", "Disable-NetAdapterPacketCoalescing -Name * -Confirm:$false -ErrorAction SilentlyContinue")

	// 3. Configure netsh global TCP low-latency/loss-prevention overrides
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "rss=enabled")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "dca=enabled")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "ecncapability=disabled")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "autotuninglevel=normal")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "heuristics=disabled")

	// 4. Inject advanced TCP socket exhaustion & RFC latency overrides
	tcpPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
	if key, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpPath, registry.SET_VALUE); err == nil {
		_ = key.SetDWordValue("MaxUserPort", 65534)
		_ = key.SetDWordValue("TcpNumConnections", 16777214)
		_ = key.SetDWordValue("Tcp1323Opts", 0)
		key.Close()
	}

	// 5. Disable dynamic Energy Efficient Ethernet, Green power savings, Flow Control & Interrupt Moderation on active NIC Class
	if classKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}`, registry.QUERY_VALUE); err == nil {
		if subKeys, err := classKey.ReadSubKeyNames(-1); err == nil {
			for _, sub := range subKeys {
				if subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}\`+sub, registry.SET_VALUE); err == nil {
					_ = subKey.SetStringValue("*EEE", "0")
					_ = subKey.SetStringValue("*GreenPower", "0")
					_ = subKey.SetStringValue("*PowerSavingMode", "0")
					_ = subKey.SetStringValue("*FlowControl", "0")
					_ = subKey.SetStringValue("*InterruptModeration", "0")
					subKey.Close()
				}
			}
		}
		classKey.Close()
	}

	return nil
}

// RestoreNetworkInterfaceSettings restores LSO, RSC, packet coalescing, and global TCP settings to OS defaults.
func RestoreNetworkInterfaceSettings() error {
	// 1. Re-enable Large Send Offload (LSO) to Windows defaults
	_ = system.Exec("powershell", "-Command", "Enable-NetAdapterLso -Name * -IPv4 -Confirm:$false -ErrorAction SilentlyContinue")
	_ = system.Exec("powershell", "-Command", "Enable-NetAdapterLso -Name * -IPv6 -Confirm:$false -ErrorAction SilentlyContinue")

	// 2. Re-enable Receive Segment Coalescing (RSC) to Windows defaults
	_ = system.Exec("powershell", "-Command", "Enable-NetAdapterRsc -Name * -Confirm:$false -ErrorAction SilentlyContinue")

	// 2b. Re-enable Packet Coalescing to Windows defaults
	_ = system.Exec("powershell", "-Command", "Enable-NetAdapterPacketCoalescing -Name * -Confirm:$false -ErrorAction SilentlyContinue")

	// 3. Restore global TCP parameters to default OS profiles
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "rss=enabled")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "dca=disabled")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "ecncapability=default")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "autotuninglevel=normal")
	_ = system.Exec("netsh", "int", "tcp", "set", "global", "heuristics=enabled")

	// 4. Restore TCP parameters from recorded baseline
	baseline, err := config.LoadBaselineState()
	if err == nil {
		tcpPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
		if key, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpPath, registry.SET_VALUE); err == nil {
			if baseline.MaxUserPortExists {
				_ = key.SetDWordValue("MaxUserPort", baseline.MaxUserPortValue)
			} else {
				_ = key.DeleteValue("MaxUserPort")
			}
			if baseline.TcpNumConnectionsExists {
				_ = key.SetDWordValue("TcpNumConnections", baseline.TcpNumConnectionsValue)
			} else {
				_ = key.DeleteValue("TcpNumConnections")
			}
			if baseline.Tcp1323OptsExists {
				_ = key.SetDWordValue("Tcp1323Opts", baseline.Tcp1323OptsValue)
			} else {
				_ = key.DeleteValue("Tcp1323Opts")
			}
			key.Close()
		}

		// 5. Restore Energy Efficient Ethernet, Green power savings, Flow Control & Interrupt Moderation
		if classKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}`, registry.QUERY_VALUE); err == nil {
			if subKeys, err := classKey.ReadSubKeyNames(-1); err == nil {
				for _, sub := range subKeys {
					if subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Class\{4d36e972-e325-11ce-bfc1-08002be10318}\`+sub, registry.SET_VALUE); err == nil {
						_ = subKey.SetStringValue("*EEE", "1")
						_ = subKey.SetStringValue("*GreenPower", "1")
						_ = subKey.SetStringValue("*PowerSavingMode", "0")
						_ = subKey.SetStringValue("*FlowControl", "3")
						_ = subKey.SetStringValue("*InterruptModeration", "1")
						subKey.Close()
					}
				}
			}
			classKey.Close()
		}
	}

	return nil
}
