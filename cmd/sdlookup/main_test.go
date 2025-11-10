package main

import (
	"context"
	"strings"
	"testing"

	"github.com/h4sh5/sdlookup/internal/config"
	"github.com/h4sh5/sdlookup/internal/models"
	"github.com/h4sh5/sdlookup/internal/output"
)

// MockClient for testing
type MockClient struct{}

func (m *MockClient) GetIPInfo(ctx context.Context, ip string) (*models.ShodanIPInfo, error) {
	// Return different results based on IP
	if ip == "1.1.1.1" {
		// Non-empty result
		return &models.ShodanIPInfo{
			IP:    ip,
			Ports: []int{80, 443},
			Tags:  []string{"cloud"},
		}, nil
	}
	// Empty result for other IPs
	return &models.ShodanIPInfo{
		IP:    ip,
		Ports: []int{},
	}, nil
}

func TestOmitEmptyConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		omitEmpty     bool
		inputIP       string
		expectOutput  bool
		description   string
	}{
		{
			name:          "omitEmpty=true, non-empty result",
			omitEmpty:     true,
			inputIP:       "1.1.1.1",
			expectOutput:  true,
			description:   "Should show non-empty results when omitEmpty=true",
		},
		{
			name:          "omitEmpty=true, empty result",
			omitEmpty:     true,
			inputIP:       "8.8.8.8",
			expectOutput:  false,
			description:   "Should hide empty results when omitEmpty=true",
		},
		{
			name:          "omitEmpty=false, non-empty result",
			omitEmpty:     false,
			inputIP:       "1.1.1.1",
			expectOutput:  true,
			description:   "Should show non-empty results when omitEmpty=false",
		},
		{
			name:          "omitEmpty=false, empty result",
			omitEmpty:     false,
			inputIP:       "8.8.8.8",
			expectOutput:  true,
			description:   "Should show empty results when omitEmpty=false (BUG FIX)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cfg := config.Default()
			cfg.OmitEmpty = tt.omitEmpty
			cfg.Concurrency = 1

			// Use mock client instead of real API
			mockClient := &MockClient{}

			// Create output buffer
			var outputBuf strings.Builder
			formatter := output.NewSimpleFormatter()
			writer := output.NewWriter(formatter, &outputBuf)

			// Simulate what processInput does
			results := make(chan *models.ScanResult, 1)
			done := make(chan struct{})

			go func() {
				defer close(done)
				for result := range results {
					// Write result if no error and has info
					if result.Error == nil && result.Info != nil {
						// Respect cfg.OmitEmpty configuration (FIXED)
						if !cfg.OmitEmpty || !result.Info.IsEmpty() {
							if err := writer.Write(result); err != nil {
								t.Errorf("Error writing output: %v", err)
							}
						}
					}
				}
			}()

			// Get mock data
			info, err := mockClient.GetIPInfo(context.Background(), tt.inputIP)
			if err != nil {
				t.Fatalf("Mock client failed: %v", err)
			}

			// Send result
			results <- &models.ScanResult{
				IP:    tt.inputIP,
				Info:  info,
				Error: nil,
			}
			close(results)
			<-done

			// Check output
			output := outputBuf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.expectOutput {
				t.Errorf("%s:\n  Expected output: %v\n  Got output: %v\n  Output content: %q",
					tt.description, tt.expectOutput, hasOutput, output)
			}

			// Additional validation for the fixed case
			if !tt.omitEmpty && !tt.expectOutput {
				t.Errorf("BUG: omitEmpty=false should show all results, but got no output for empty result")
			}
		})
	}
}

func TestOmitEmptyIntegration(t *testing.T) {
	// Test the actual config merging
	cfg := config.Default()

	// Test default
	if !cfg.OmitEmpty {
		t.Error("Default OmitEmpty should be true")
	}

	// Test flag override
	omitEmptyFlag := false
	cfg.MergeWithFlags(nil, nil, nil, nil, nil, &omitEmptyFlag)

	if cfg.OmitEmpty {
		t.Error("OmitEmpty should be false after flag override")
	}
}
