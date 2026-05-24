package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

// GameSubnet defines a list of subnets associated with a game.
type GameSubnet struct {
	Name    string   `json:"name"`
	Subnets []string `json:"subnets"`
}

// GameConfig matches the JSON schema of games.json.
type GameConfig struct {
	Games []GameSubnet `json:"games"`
}

// Embedded default subnets to act as a foolproof fallback if games.json is deleted.
var DefaultGameSubnets = []GameSubnet{
	{
		Name: "Riot Games Hubs (Valorant, LoL)",
		Subnets: []string{
			"162.249.72.0/22",
			"192.207.0.0/16",
		},
	},
	{
		Name: "Valve Esports Hubs (CS2, Dota 2)",
		Subnets: []string{
			"155.133.0.0/16",
			"162.254.0.0/16",
		},
	},
}

// LoadGamesConfig loads the games subnets from the config folder or returns default fallbacks.
func LoadGamesConfig() ([]GameSubnet, error) {
	data, err := os.ReadFile(`internal/config/games.json`)
	if err != nil {
		return DefaultGameSubnets, nil // Graceful fallback
	}

	var config GameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultGameSubnets, fmt.Errorf("failed to parse games.json: %w (falling back to defaults)", err)
	}

	return config.Games, nil
}

// ConvertCIDRToDottedQuad translates CIDR subnets like '192.168.1.0/24'
// to separate Subnet IP '192.168.1.0' and Mask '255.255.255.0' strings.
func ConvertCIDRToDottedQuad(cidr string) (string, string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("invalid CIDR block %s: %w", cidr, err)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return "", "", fmt.Errorf("CIDR %s is not an IPv4 address", cidr)
	}

	maskBytes := ipnet.Mask
	if len(maskBytes) != 4 {
		return "", "", fmt.Errorf("invalid IPv4 mask length: %d", len(maskBytes))
	}

	maskStr := fmt.Sprintf("%d.%d.%d.%d", maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])
	return ipv4.String(), maskStr, nil
}

// InjectGameRoutes discovers the active gateway and injects optimized lowest-metric routes for all subnets.
// It returns a list of successfully injected subnets to allow exact cleanup.
func InjectGameRoutes() ([]string, error) {
	nic, err := DiscoverActiveNIC()
	if err != nil {
		return nil, fmt.Errorf("cannot inject routes: failed to discover active gateway: %w", err)
	}

	if nic.DefaultGateway == "" {
		return nil, fmt.Errorf("cannot inject routes: default gateway is empty for active interface %s", nic.GUID)
	}

	games, err := LoadGamesConfig()
	if err != nil {
		// Log warning but continue with defaults
		fmt.Printf("⚠️  Warning loading games config: %v. Using defaults.\n", err)
	}

	var injected []string
	for _, game := range games {
		for _, subnet := range game.Subnets {
			ip, mask, err := ConvertCIDRToDottedQuad(subnet)
			if err != nil {
				continue
			}

			// Add route via route add command
			// route add [Subnet] mask [Mask] [Gateway] metric 1
			cmd := exec.Command("route", "add", ip, "mask", mask, nic.DefaultGateway, "metric", "1")
			if err := cmd.Run(); err == nil {
				injected = append(injected, subnet)
			}
		}
	}

	return injected, nil
}

// DeleteGameRoutes loop-deletes the injected custom gaming subnets to restore default system routing paths.
func DeleteGameRoutes(subnets []string) int {
	deletedCount := 0
	for _, subnet := range subnets {
		// Extract IP from CIDR (e.g. 192.168.1.0/24 -> 192.168.1.0)
		parts := strings.Split(subnet, "/")
		if len(parts) == 0 {
			continue
		}
		ip := parts[0]

		// Execute route delete [SubnetIP]
		cmd := exec.Command("route", "delete", ip)
		if err := cmd.Run(); err == nil {
			deletedCount++
		}
	}
	return deletedCount
}
