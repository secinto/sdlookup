package output

import (
	"testing"

	"github.com/h4sh5/sdlookup/internal/models"
)

func BenchmarkCSVFormatter_Format(b *testing.B) {
	result := &models.ScanResult{
		IP: "192.168.1.1",
		Info: &models.ShodanIPInfo{
			IP:        "192.168.1.1",
			Ports:     []int{80, 443, 8080, 3000, 5000},
			Hostnames: []string{"example.com", "test.example.com"},
			Tags:      []string{"cloud", "web"},
			Cpes:      []string{"cpe:/a:vendor:product:1.0"},
			Vulns:     []string{"CVE-2021-1234", "CVE-2021-5678"},
		},
	}

	formatter := NewCSVFormatter(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(result)
	}
}

func BenchmarkJSONFormatter_Format(b *testing.B) {
	result := &models.ScanResult{
		IP: "192.168.1.1",
		Info: &models.ShodanIPInfo{
			IP:        "192.168.1.1",
			Ports:     []int{80, 443, 8080, 3000, 5000},
			Hostnames: []string{"example.com", "test.example.com"},
			Tags:      []string{"cloud", "web"},
			Cpes:      []string{"cpe:/a:vendor:product:1.0"},
			Vulns:     []string{"CVE-2021-1234", "CVE-2021-5678"},
		},
	}

	formatter := NewJSONFormatter(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(result)
	}
}

func BenchmarkSimpleFormatter_Format(b *testing.B) {
	result := &models.ScanResult{
		IP: "192.168.1.1",
		Info: &models.ShodanIPInfo{
			IP:    "192.168.1.1",
			Ports: []int{80, 443, 8080, 3000, 5000},
		},
	}

	formatter := NewSimpleFormatter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(result)
	}
}

func BenchmarkCSVFormatter_FormatBatch(b *testing.B) {
	results := make([]*models.ScanResult, 100)
	for i := 0; i < 100; i++ {
		results[i] = &models.ScanResult{
			IP: "192.168.1.1",
			Info: &models.ShodanIPInfo{
				IP:        "192.168.1.1",
				Ports:     []int{80, 443},
				Hostnames: []string{"example.com"},
			},
		}
	}

	formatter := NewCSVFormatter(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.FormatBatch(results)
	}
}
