package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Concurrency != 10 {
		t.Errorf("Default concurrency = %d, want 10", cfg.Concurrency)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Default timeout = %v, want 30s", cfg.Timeout)
	}

	if cfg.API.BaseURL != "https://internetdb.shodan.io" {
		t.Errorf("Default API URL = %s, want https://internetdb.shodan.io", cfg.API.BaseURL)
	}

	if !cfg.API.VerifyTLS {
		t.Error("Default VerifyTLS should be true")
	}

	if !cfg.Cache.Enabled {
		t.Error("Default cache should be enabled")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	configData := `
concurrency: 20
timeout: 60s
rate_limit: 100
omit_empty: false
api:
  base_url: https://test.example.com
  verify_tls: false
  max_retries: 5
output:
  format: json
  show_services: true
cache:
  enabled: false
`

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if cfg.Concurrency != 20 {
		t.Errorf("Loaded concurrency = %d, want 20", cfg.Concurrency)
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Loaded timeout = %v, want 60s", cfg.Timeout)
	}

	if cfg.API.BaseURL != "https://test.example.com" {
		t.Errorf("Loaded API URL = %s, want https://test.example.com", cfg.API.BaseURL)
	}

	if cfg.API.VerifyTLS {
		t.Error("Loaded VerifyTLS should be false")
	}

	if cfg.Cache.Enabled {
		t.Error("Loaded cache should be disabled")
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Loaded output format = %s, want json", cfg.Output.Format)
	}
}

func TestMergeWithFlags(t *testing.T) {
	cfg := Default()

	concurrency := 50
	jsonOutput := true
	csvOutput := false
	onlyHost := false
	servicesOutput := true
	omitEmpty := false

	cfg.MergeWithFlags(&concurrency, &jsonOutput, &csvOutput, &onlyHost, &servicesOutput, &omitEmpty)

	if cfg.Concurrency != 50 {
		t.Errorf("Merged concurrency = %d, want 50", cfg.Concurrency)
	}

	if cfg.Output.Format != "json" {
		t.Errorf("Merged format = %s, want json", cfg.Output.Format)
	}

	if !cfg.Output.ShowServices {
		t.Error("Merged ShowServices should be true")
	}

	if cfg.OmitEmpty {
		t.Error("Merged OmitEmpty should be false")
	}
}

func TestMergeWithFlags_SimpleFormat(t *testing.T) {
	cfg := Default()

	onlyHost := true
	cfg.MergeWithFlags(nil, nil, nil, &onlyHost, nil, nil)

	if cfg.Output.Format != "simple" {
		t.Errorf("Format with onlyHost = %s, want simple", cfg.Output.Format)
	}
}
