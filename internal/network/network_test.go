package network

import (
	"encoding/json"
	"testing"
)

func TestCIDRToDottedQuadConversion(t *testing.T) {
	tests := []struct {
		cidr         string
		expectedIP   string
		expectedMask string
		expectError  bool
	}{
		{"192.168.1.0/24", "192.168.1.0", "255.255.255.0", false},
		{"10.0.0.0/8", "10.0.0.0", "255.0.0.0", false},
		{"162.249.72.0/22", "162.249.72.0", "255.255.252.0", false},
		{"invalid-cidr", "", "", true},
		{"2001:db8::/32", "", "", true}, // IPv6 is excluded by design
	}

	for _, tc := range tests {
		ip, mask, err := ConvertCIDRToDottedQuad(tc.cidr)
		if tc.expectError {
			if err == nil {
				t.Errorf("expected error for CIDR %q, got nil", tc.cidr)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for CIDR %q: %v", tc.cidr, err)
			}
			if ip != tc.expectedIP {
				t.Errorf("IP mismatch for %q: expected %q, got %q", tc.cidr, tc.expectedIP, ip)
			}
			if mask != tc.expectedMask {
				t.Errorf("mask mismatch for %q: expected %q, got %q", tc.cidr, tc.expectedMask, mask)
			}
		}
	}
}

func TestGamesJSONSchemaSerialization(t *testing.T) {
	// Sample JSON parsing test
	sampleJSON := `{
		"games": [
			{
				"name": "Riot Hub",
				"subnets": ["162.249.72.0/22"]
			}
		]
	}`

	var config GameConfig
	err := json.Unmarshal([]byte(sampleJSON), &config)
	if err != nil {
		t.Fatalf("failed to deserialize sample JSON: %v", err)
	}

	if len(config.Games) != 1 || config.Games[0].Name != "Riot Hub" {
		t.Errorf("deserialization name mismatch")
	}

	if len(config.Games[0].Subnets) != 1 || config.Games[0].Subnets[0] != "162.249.72.0/22" {
		t.Errorf("deserialization subnets mismatch")
	}
}

func TestActiveNICDiscoveryCompileBoundaries(t *testing.T) {
	// Verify that ActiveNICInfo struct compiles cleanly
	info := ActiveNICInfo{
		GUID:           "{GUID}",
		IPAddress:      "192.168.1.5",
		DefaultGateway: "192.168.1.1",
	}

	if info.GUID != "{GUID}" {
		t.Errorf("struct compile parameters mismatch")
	}
}
