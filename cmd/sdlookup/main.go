package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/h4sh5/sdlookup/internal/api"
	"github.com/h4sh5/sdlookup/internal/config"
	"github.com/h4sh5/sdlookup/internal/models"
	"github.com/h4sh5/sdlookup/internal/output"
	"github.com/h4sh5/sdlookup/internal/scanner"
	"github.com/h4sh5/sdlookup/pkg/validator"
)

var (
	// Command-line flags
	concurrency    = flag.Int("c", 0, "Set the concurrency level (default: from config or 10)")
	jsonOutput     = flag.Bool("json", false, "Show output as JSON format")
	csvOutput      = flag.Bool("csv", false, "Show output as CSV format (default)")
	servicesOutput = flag.Bool("services", false, "Create services.json output file")
	onlyHost       = flag.Bool("open", false, "Show output as 'IP:Port' only")
	omitEmpty      = flag.Bool("omitEmpty", true, "Only provide output if an entry exists")
	configFile     = flag.String("config", "", "Path to configuration file")
	verbose        = flag.Bool("v", false, "Verbose output (debug logging)")
	noProgress     = flag.Bool("no-progress", false, "Disable progress indication")
	noCache        = flag.Bool("no-cache", false, "Disable caching")
	version        = flag.Bool("version", false, "Show version information")
)

const (
	appVersion = "2.0.0"
	appName    = "sdlookup"
)

func main() {
	flag.Parse()

	// Show version and exit
	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down gracefully...")
		cancel()
	}()

	// Load configuration
	cfg := loadConfig()

	// Merge with command-line flags
	cfg.MergeWithFlags(concurrency, jsonOutput, csvOutput, onlyHost, servicesOutput, omitEmpty)

	// Setup logger
	logger := setupLogger(*verbose)

	// Create API client
	var cache api.Cache
	if cfg.Cache.Enabled && !*noCache {
		cache = api.NewLRUCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
		logger.Debug("cache enabled", "max_size", cfg.Cache.MaxSize, "ttl", cfg.Cache.TTL)
	}

	client := api.NewClient(
		cfg.Timeout,
		api.WithBaseURL(cfg.API.BaseURL),
		api.WithLogger(logger),
		api.WithVerifyTLS(cfg.API.VerifyTLS),
		api.WithRateLimit(cfg.RateLimit),
		api.WithCache(cache),
		api.WithRetries(cfg.API.MaxRetries, cfg.API.RetryBackoff),
		api.WithConcurrency(cfg.Concurrency),
	)

	// Create scanner
	scan := scanner.NewScanner(client, logger, cfg.OmitEmpty)

	// Create output formatter
	formatter := createFormatter(cfg.Output.Format, *onlyHost)
	writer := output.NewWriter(formatter, os.Stdout)

	// Create services collector
	servicesCollector := output.NewServicesCollector(cfg.Output.ShowServices)

	// Check if input is from stdin or arguments
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// Input from arguments
		args := flag.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: sdlookup [options] <IP|CIDR>")
			fmt.Fprintln(os.Stderr, "   or: echo <IP|CIDR> | sdlookup [options]")
			flag.PrintDefaults()
			os.Exit(1)
		}

		for _, arg := range args {
			if err := processInput(ctx, scan, writer, servicesCollector, arg, cfg); err != nil {
				logger.Error("failed to process input", "input", arg, "error", err)
			}
		}
	} else {
		// Input from stdin
		if err := processStdin(ctx, scan, writer, servicesCollector, cfg); err != nil {
			logger.Error("failed to process stdin", "error", err)
			os.Exit(1)
		}
	}

	// Write services file if enabled
	if cfg.Output.ShowServices && servicesCollector.Count() > 0 {
		servicesFile := "services.json"
		if err := servicesCollector.WriteToFile(servicesFile); err != nil {
			logger.Error("failed to write services file", "error", err)
		} else {
			logger.Info("services file written", "file", servicesFile, "count", servicesCollector.Count())
		}
	}
}

// loadConfig loads configuration from file or returns default
func loadConfig() *config.Config {
	// Try to load from specified config file
	if *configFile != "" {
		cfg, err := config.LoadFromFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config file %s: %v\n", *configFile, err)
			return config.Default()
		}
		return cfg
	}

	// Try to load from default locations
	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".sdlookup.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			cfg, err := config.LoadFromFile(defaultPath)
			if err == nil {
				return cfg
			}
		}
	}

	return config.Default()
}

// setupLogger creates a logger with appropriate level
func setupLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// createFormatter creates the appropriate output formatter
func createFormatter(format string, onlyIPPort bool) output.Formatter {
	switch format {
	case "json":
		return output.NewJSONFormatter(false)
	case "simple":
		return output.NewSimpleFormatter()
	case "csv":
		fallthrough
	default:
		return output.NewCSVFormatter(onlyIPPort)
	}
}

// processInput processes a single input (IP or CIDR)
func processInput(
	ctx context.Context,
	scan *scanner.Scanner,
	writer *output.Writer,
	servicesCollector *output.ServicesCollector,
	input string,
	cfg *config.Config,
) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Create results channel
	results := make(chan *models.ScanResult, cfg.Concurrency)

	// Process results in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for result := range results {
			if result.Error == nil && result.Info != nil && !result.Info.IsEmpty() {
				if err := writer.Write(result); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				}
				servicesCollector.Add(result)
			}
		}
	}()

	// Scan the input
	err := scan.ScanInput(ctx, input, cfg.Concurrency, results)
	close(results)
	<-done

	return err
}

// processStdin processes input from stdin
func processStdin(
	ctx context.Context,
	scan *scanner.Scanner,
	writer *output.Writer,
	servicesCollector *output.ServicesCollector,
	cfg *config.Config,
) error {
	// Read all inputs first to calculate total for progress
	var inputs []string
	stdinScanner := bufio.NewScanner(os.Stdin)
	for stdinScanner.Scan() {
		line := strings.TrimSpace(stdinScanner.Text())
		if line != "" {
			inputs = append(inputs, line)
		}
	}

	if err := stdinScanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	if len(inputs) == 0 {
		return nil
	}

	// Estimate total IPs for progress tracking
	totalIPs := estimateTotalIPs(inputs)

	// Create progress tracker
	showProgress := cfg.Output.ShowProgress && !*noProgress
	progress := scanner.NewProgress(totalIPs, os.Stderr, showProgress)

	// Create results channel
	results := make(chan *models.ScanResult, cfg.Concurrency)

	// Process results in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for result := range results {
			success := result.Error == nil && result.Info != nil && !result.Info.IsEmpty()
			progress.Increment(success)

			if success {
				if err := writer.Write(result); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
				}
				servicesCollector.Add(result)
			}
		}
	}()

	// Process all inputs
	for _, input := range inputs {
		if err := scan.ScanInput(ctx, input, cfg.Concurrency, results); err != nil {
			if ctx.Err() != nil {
				break // Context cancelled
			}
		}
	}

	close(results)
	<-done
	progress.Done()

	return ctx.Err()
}

// estimateTotalIPs estimates the total number of IPs to scan
func estimateTotalIPs(inputs []string) int {
	total := 0
	for _, input := range inputs {
		if validator.IsCIDR(input) {
			// Rough estimate - count actual IPs
			ips, err := validator.CountIPsInCIDR(input)
			if err == nil {
				total += ips
			} else {
				total++ // Fallback to 1
			}
		} else {
			total++
		}
	}
	return total
}
