package validator

import (
	"testing"
)

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		wantError bool
	}{
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv4 public", "8.8.8.8", false},
		{"valid IPv6", "2001:db8::1", false},
		{"valid IPv6 localhost", "::1", false},
		{"invalid IP", "256.1.1.1", true},
		{"invalid format", "not.an.ip", true},
		{"empty string", "", true},
		{"spaces only", "   ", true},
		{"IPv4 with spaces", " 192.168.1.1 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateIP(%q) error = %v, wantError %v", tt.ip, err, tt.wantError)
			}
		})
	}
}

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		wantError bool
	}{
		{"valid CIDR /24", "192.168.1.0/24", false},
		{"valid CIDR /32", "192.168.1.1/32", false},
		{"valid CIDR /16", "10.0.0.0/16", false},
		{"valid IPv6 CIDR", "2001:db8::/32", false},
		{"invalid CIDR no mask", "192.168.1.0", true},
		{"invalid CIDR bad mask", "192.168.1.0/33", true},
		{"invalid CIDR bad IP", "256.1.1.0/24", true},
		{"empty string", "", true},
		{"CIDR with spaces", " 192.168.1.0/24 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCIDR(tt.cidr)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateCIDR(%q) error = %v, wantError %v", tt.cidr, err, tt.wantError)
			}
		})
	}
}

func TestIsCIDR(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"192.168.1.0/24", true},
		{"192.168.1.1/32", true},
		{"192.168.1.1", false},
		{"not.a.cidr", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsCIDR(tt.input); got != tt.expected {
				t.Errorf("IsCIDR(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsIP(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"192.168.1.1", true},
		{"8.8.8.8", true},
		{"2001:db8::1", true},
		{"256.1.1.1", false},
		{"not.an.ip", false},
		{"", false},
		{" 192.168.1.1 ", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsIP(tt.input); got != tt.expected {
				t.Errorf("IsIP(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCountIPsInCIDR(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		expected  int
		wantError bool
	}{
		{"IPv4 /32", "192.168.1.1/32", 1, false},
		{"IPv4 /24", "192.168.1.0/24", 256, false},
		{"IPv4 /16", "10.0.0.0/16", 65536, false},
		{"IPv4 /8", "10.0.0.0/8", 16777216, false},
		{"IPv6 /128", "2001:db8::1/128", 1, false},
		{"IPv6 /64", "2001:db8::/64", 1<<20, false}, // Capped
		{"invalid CIDR", "not.a.cidr", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := CountIPsInCIDR(tt.cidr)
			if (err != nil) != tt.wantError {
				t.Errorf("CountIPsInCIDR(%q) error = %v, wantError %v", tt.cidr, err, tt.wantError)
				return
			}
			if count != tt.expected {
				t.Errorf("CountIPsInCIDR(%q) = %d, want %d", tt.cidr, count, tt.expected)
			}
		})
	}
}
