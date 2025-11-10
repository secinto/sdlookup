package models

import (
	"testing"
)

func TestShodanIPInfo_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		info     *ShodanIPInfo
		expected bool
	}{
		{
			name: "empty info",
			info: &ShodanIPInfo{
				IP: "1.2.3.4",
			},
			expected: true,
		},
		{
			name: "has ports",
			info: &ShodanIPInfo{
				IP:    "1.2.3.4",
				Ports: []int{80, 443},
			},
			expected: false,
		},
		{
			name: "has hostnames",
			info: &ShodanIPInfo{
				IP:        "1.2.3.4",
				Hostnames: []string{"example.com"},
			},
			expected: false,
		},
		{
			name: "has vulns",
			info: &ShodanIPInfo{
				IP:    "1.2.3.4",
				Vulns: []string{"CVE-2021-1234"},
			},
			expected: false,
		},
		{
			name: "has all fields",
			info: &ShodanIPInfo{
				IP:        "1.2.3.4",
				Ports:     []int{80},
				Hostnames: []string{"example.com"},
				Vulns:     []string{"CVE-2021-1234"},
				Tags:      []string{"cloud"},
				Cpes:      []string{"cpe:/a:vendor:product"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}
