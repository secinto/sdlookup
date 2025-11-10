package output

import (
	"strings"
	"testing"

	"github.com/h4sh5/sdlookup/internal/models"
)

func TestCSVFormatter_Format(t *testing.T) {
	tests := []struct {
		name       string
		result     *models.ScanResult
		onlyIPPort bool
		expected   string
		wantError  bool
	}{
		{
			name: "simple IP:Port format",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{80, 443},
				},
			},
			onlyIPPort: true,
			expected:   "192.168.1.1:80\n192.168.1.1:443",
			wantError:  false,
		},
		{
			name: "full CSV format",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:        "192.168.1.1",
					Ports:     []int{80},
					Hostnames: []string{"example.com"},
					Tags:      []string{"cloud"},
					Cpes:      []string{"cpe:/a:vendor:product"},
					Vulns:     []string{"CVE-2021-1234"},
				},
			},
			onlyIPPort: false,
			expected:   "192.168.1.1:80,example.com,cloud,cpe:/a:vendor:product,CVE-2021-1234",
			wantError:  false,
		},
		{
			name: "empty result - simple format",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{},
				},
			},
			onlyIPPort: true,
			expected:   "192.168.1.1",
			wantError:  false,
		},
		{
			name: "empty result - full CSV format",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{},
				},
			},
			onlyIPPort: false,
			expected:   "192.168.1.1,(no ports),,,,",
			wantError:  false,
		},
		{
			name: "error result",
			result: &models.ScanResult{
				IP:    "192.168.1.1",
				Error: &testError{"test error"},
			},
			onlyIPPort: true,
			expected:   "",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewCSVFormatter(tt.onlyIPPort)
			output, err := formatter.Format(tt.result)

			if (err != nil) != tt.wantError {
				t.Errorf("Format() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if output != tt.expected {
				t.Errorf("Format() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	result := &models.ScanResult{
		IP: "192.168.1.1",
		Info: &models.ShodanIPInfo{
			IP:    "192.168.1.1",
			Ports: []int{80, 443},
		},
	}

	formatter := NewJSONFormatter(false)
	output, err := formatter.Format(result)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Check that it contains expected JSON fields
	if !strings.Contains(output, `"ip":"192.168.1.1"`) {
		t.Errorf("Output missing IP field: %s", output)
	}
	if !strings.Contains(output, `"ports":[80,443]`) {
		t.Errorf("Output missing ports field: %s", output)
	}
}

func TestSimpleFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		result   *models.ScanResult
		expected string
	}{
		{
			name: "with ports",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{80, 443, 8080},
				},
			},
			expected: "192.168.1.1:80\n192.168.1.1:443\n192.168.1.1:8080",
		},
		{
			name: "empty ports (omitEmpty=false case)",
			result: &models.ScanResult{
				IP: "192.168.1.1",
				Info: &models.ShodanIPInfo{
					IP:    "192.168.1.1",
					Ports: []int{},
				},
			},
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewSimpleFormatter()
			output, err := formatter.Format(tt.result)

			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			if output != tt.expected {
				t.Errorf("Format() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestPortConversionFix(t *testing.T) {
	// This test verifies the bug fix for port conversion
	// Previously: string(rune(port)) converted 80 to "P"
	// Now: Should be "80"

	result := &models.ScanResult{
		IP: "192.168.1.1",
		Info: &models.ShodanIPInfo{
			IP:    "192.168.1.1",
			Ports: []int{80, 443, 8080},
		},
	}

	formatter := NewSimpleFormatter()
	output, err := formatter.Format(result)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Should contain actual port numbers, not unicode characters
	if strings.Contains(output, "P") && !strings.Contains(output, "80") {
		t.Errorf("Port conversion bug still exists! Output: %q", output)
	}

	if !strings.Contains(output, "80") {
		t.Errorf("Output missing port 80: %q", output)
	}
	if !strings.Contains(output, "443") {
		t.Errorf("Output missing port 443: %q", output)
	}
	if !strings.Contains(output, "8080") {
		t.Errorf("Output missing port 8080: %q", output)
	}
}

// testError implements error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
