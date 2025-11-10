package models

// ShodanIPInfo represents the response from Shodan's InternetDB API
type ShodanIPInfo struct {
	Cpes      []string `json:"cpes"`
	Hostnames []string `json:"hostnames"`
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
	Tags      []string `json:"tags"`
	Vulns     []string `json:"vulns"`
}

// IsEmpty returns true if the ShodanIPInfo has no meaningful data
func (s *ShodanIPInfo) IsEmpty() bool {
	return len(s.Ports) == 0 && len(s.Hostnames) == 0 && len(s.Vulns) == 0
}

// Service represents a network service on a host
type Service struct {
	IP       string `json:"ip"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Service  string `json:"service,omitempty"`
}

// ScanResult represents a complete scan result for an IP
type ScanResult struct {
	IP       string
	Info     *ShodanIPInfo
	Error    error
	Services []Service
}
