package validator

import (
	"fmt"
	"net"
	"strings"
)

// ValidateIP validates an IP address
func ValidateIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return fmt.Errorf("empty IP address")
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	return nil
}

// ValidateCIDR validates a CIDR notation
func ValidateCIDR(cidr string) error {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return fmt.Errorf("empty CIDR")
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %s: %w", cidr, err)
	}

	return nil
}

// IsCIDR checks if a string is in CIDR notation
func IsCIDR(input string) bool {
	_, _, err := net.ParseCIDR(strings.TrimSpace(input))
	return err == nil
}

// IsIP checks if a string is a valid IP address
func IsIP(input string) bool {
	return net.ParseIP(strings.TrimSpace(input)) != nil
}

// CountIPsInCIDR returns the number of IPs in a CIDR range
func CountIPsInCIDR(cidr string) (int, error) {
	_, ipnet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return 0, err
	}

	ones, bits := ipnet.Mask.Size()
	if bits == 32 {
		// IPv4
		return 1 << uint(bits-ones), nil
	}
	// IPv6 - cap at a reasonable number to avoid overflow
	if bits-ones > 20 {
		return 1 << 20, nil // Cap at ~1M for display purposes
	}
	return 1 << uint(bits-ones), nil
}
