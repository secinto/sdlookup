package output

import (
	"fmt"
	"os"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/h4sh5/sdlookup/internal/models"
)

// ServicesCollector collects services in a thread-safe manner
type ServicesCollector struct {
	mu       sync.Mutex
	services []models.Service
	enabled  bool
}

// NewServicesCollector creates a new services collector
func NewServicesCollector(enabled bool) *ServicesCollector {
	return &ServicesCollector{
		services: make([]models.Service, 0),
		enabled:  enabled,
	}
}

// Add adds services from a scan result
func (s *ServicesCollector) Add(result *models.ScanResult) {
	if !s.enabled || result.Error != nil || result.Info == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, port := range result.Info.Ports {
		s.services = append(s.services, models.Service{
			IP:       result.IP,
			Protocol: "tcp",
			Port:     port,
			Service:  "", // Service name unknown from InternetDB
		})
	}
}

// WriteToFile writes collected services to a JSON file
func (s *ServicesCollector) WriteToFile(filename string) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.services) == 0 {
		return nil
	}

	data, err := jsoniter.MarshalIndent(s.services, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling services: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing services file: %w", err)
	}

	return nil
}

// Count returns the number of collected services
func (s *ServicesCollector) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.services)
}

// Services returns a copy of collected services
func (s *ServicesCollector) Services() []models.Service {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]models.Service, len(s.services))
	copy(result, s.services)
	return result
}
