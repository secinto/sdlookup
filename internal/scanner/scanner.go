package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/projectdiscovery/mapcidr"
	"github.com/h4sh5/sdlookup/internal/api"
	"github.com/h4sh5/sdlookup/internal/models"
	"github.com/h4sh5/sdlookup/pkg/validator"
)

// Scanner manages IP scanning operations
type Scanner struct {
	client    *api.Client
	logger    *slog.Logger
	omitEmpty bool
}

// NewScanner creates a new scanner
func NewScanner(client *api.Client, logger *slog.Logger, omitEmpty bool) *Scanner {
	return &Scanner{
		client:    client,
		logger:    logger,
		omitEmpty: omitEmpty,
	}
}

// ScanIP scans a single IP address
func (s *Scanner) ScanIP(ctx context.Context, ip string) (*models.ScanResult, error) {
	// Validate IP
	if err := validator.ValidateIP(ip); err != nil {
		s.logger.Warn("invalid IP address", "ip", ip, "error", err)
		return &models.ScanResult{
			IP:    ip,
			Error: err,
		}, err
	}

	info, err := s.client.GetIPInfo(ctx, ip)
	if err != nil {
		s.logger.Error("failed to get IP info", "ip", ip, "error", err)
		return &models.ScanResult{
			IP:    ip,
			Error: err,
		}, err
	}

	// Log if result is empty (filtering happens at output layer)
	if s.omitEmpty && info.IsEmpty() {
		s.logger.Debug("result is empty", "ip", ip)
	}

	return &models.ScanResult{
		IP:   ip,
		Info: info,
	}, nil
}

// ScanCIDR scans all IPs in a CIDR range
func (s *Scanner) ScanCIDR(ctx context.Context, cidr string, concurrency int, results chan<- *models.ScanResult) error {
	// Validate CIDR
	if err := validator.ValidateCIDR(cidr); err != nil {
		s.logger.Warn("invalid CIDR", "cidr", cidr, "error", err)
		return err
	}

	// Expand CIDR to IPs
	ips, err := mapcidr.IPAddresses(cidr)
	if err != nil {
		s.logger.Error("failed to expand CIDR", "cidr", cidr, "error", err)
		return fmt.Errorf("expanding CIDR %s: %w", cidr, err)
	}

	s.logger.Info("scanning CIDR range", "cidr", cidr, "count", len(ips), "concurrency", concurrency)

	// Scan IPs concurrently
	return s.ScanIPs(ctx, ips, concurrency, results)
}

// ScanIPs scans multiple IPs concurrently
func (s *Scanner) ScanIPs(ctx context.Context, ips []string, concurrency int, results chan<- *models.ScanResult) error {
	var wg sync.WaitGroup
	ipChan := make(chan string, concurrency)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for ip := range ipChan {
				select {
				case <-ctx.Done():
					s.logger.Debug("worker cancelled", "worker", workerID)
					return
				default:
					result, _ := s.ScanIP(ctx, ip)
					if result != nil {
						select {
						case results <- result:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}(i)
	}

	// Feed IPs to workers
	go func() {
		for _, ip := range ips {
			select {
			case ipChan <- ip:
			case <-ctx.Done():
				close(ipChan)
				return
			}
		}
		close(ipChan)
	}()

	wg.Wait()
	return ctx.Err()
}

// ScanInput scans an input string (IP or CIDR)
func (s *Scanner) ScanInput(ctx context.Context, input string, concurrency int, results chan<- *models.ScanResult) error {
	if validator.IsCIDR(input) {
		return s.ScanCIDR(ctx, input, concurrency, results)
	}

	result, err := s.ScanIP(ctx, input)
	if result != nil {
		select {
		case results <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}
