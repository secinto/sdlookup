package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	Concurrency int           `yaml:"concurrency"`
	Timeout     time.Duration `yaml:"timeout"`
	RateLimit   int           `yaml:"rate_limit"` // requests per minute
	OmitEmpty   bool          `yaml:"omit_empty"`
	API         APIConfig     `yaml:"api"`
	Output      OutputConfig  `yaml:"output"`
	Cache       CacheConfig   `yaml:"cache"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	BaseURL         string `yaml:"base_url"`
	UserAgent       string `yaml:"user_agent"`
	VerifyTLS       bool   `yaml:"verify_tls"`
	MaxRetries      int    `yaml:"max_retries"`
	RetryBackoff    time.Duration `yaml:"retry_backoff"`
}

// OutputConfig holds output-related configuration
type OutputConfig struct {
	Format       string `yaml:"format"` // csv, json, simple
	ShowServices bool   `yaml:"show_services"`
	FilePath     string `yaml:"file_path,omitempty"`
	ShowProgress bool   `yaml:"show_progress"`
}

// CacheConfig holds cache-related configuration
type CacheConfig struct {
	Enabled bool          `yaml:"enabled"`
	TTL     time.Duration `yaml:"ttl"`
	MaxSize int           `yaml:"max_size"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Concurrency: 10,
		Timeout:     30 * time.Second,
		RateLimit:   60, // 60 requests per minute
		OmitEmpty:   true,
		API: APIConfig{
			BaseURL:      "https://internetdb.shodan.io",
			UserAgent:    "sdlookup/2.0",
			VerifyTLS:    true,
			MaxRetries:   3,
			RetryBackoff: 2 * time.Second,
		},
		Output: OutputConfig{
			Format:       "csv",
			ShowServices: false,
			ShowProgress: true,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     24 * time.Hour,
			MaxSize: 10000,
		},
	}
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// MergeWithFlags merges configuration with command-line flags
func (c *Config) MergeWithFlags(
	concurrency *int,
	jsonOutput *bool,
	csvOutput *bool,
	onlyHost *bool,
	servicesOutput *bool,
	omitEmpty *bool,
) {
	if concurrency != nil && *concurrency > 0 {
		c.Concurrency = *concurrency
	}
	if jsonOutput != nil && *jsonOutput {
		c.Output.Format = "json"
	}
	if csvOutput != nil && *csvOutput {
		c.Output.Format = "csv"
	}
	if onlyHost != nil && *onlyHost {
		c.Output.Format = "simple"
	}
	if servicesOutput != nil {
		c.Output.ShowServices = *servicesOutput
	}
	if omitEmpty != nil {
		c.OmitEmpty = *omitEmpty
	}
}
