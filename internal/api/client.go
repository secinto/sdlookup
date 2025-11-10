package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/h4sh5/sdlookup/internal/models"
)

// Client represents a Shodan InternetDB API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	limiter    *RateLimiter
	cache      Cache
	maxRetries int
	backoff    time.Duration
}

// Cache interface for caching API responses
type Cache interface {
	Get(key string) (*models.ShodanIPInfo, bool)
	Set(key string, value *models.ShodanIPInfo)
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithBaseURL sets the base URL for the API
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithVerifyTLS controls TLS certificate verification
func WithVerifyTLS(verify bool) ClientOption {
	return func(c *Client) {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.InsecureSkipVerify = !verify
		}
	}
}

// WithRateLimit sets the rate limit (requests per minute)
func WithRateLimit(requestsPerMinute int) ClientOption {
	return func(c *Client) {
		c.limiter = NewRateLimiter(requestsPerMinute)
	}
}

// WithCache sets the cache
func WithCache(cache Cache) ClientOption {
	return func(c *Client) {
		c.cache = cache
	}
}

// WithRetries sets retry configuration
func WithRetries(maxRetries int, backoff time.Duration) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.backoff = backoff
	}
}

// WithConcurrency sets the maximum concurrent connections
func WithConcurrency(maxConns int) ClientOption {
	return func(c *Client) {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.MaxIdleConns = maxConns * 2
			transport.MaxIdleConnsPerHost = maxConns
		}
	}
}

// NewClient creates a new Shodan InternetDB API client
func NewClient(timeout time.Duration, opts ...ClientOption) *Client {
	client := &Client{
		baseURL: "https://internetdb.shodan.io",
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
		logger:     slog.Default(),
		limiter:    NewRateLimiter(60), // default 60 req/min
		maxRetries: 3,
		backoff:    2 * time.Second,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// GetIPInfo fetches information for a single IP address
func (c *Client) GetIPInfo(ctx context.Context, ip string) (*models.ShodanIPInfo, error) {
	// Check cache first
	if c.cache != nil {
		if info, ok := c.cache.Get(ip); ok {
			c.logger.Debug("cache hit", "ip", ip)
			return info, nil
		}
	}

	var info *models.ShodanIPInfo
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := c.backoff * time.Duration(1<<uint(attempt-1))
			c.logger.Debug("retrying after backoff", "ip", ip, "attempt", attempt, "backoff", backoffDuration)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		// Wait for rate limiter
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		info, lastErr = c.fetchIPInfo(ctx, ip)
		if lastErr == nil {
			// Cache successful result
			if c.cache != nil {
				c.cache.Set(ip, info)
			}
			return info, nil
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		c.logger.Warn("API request failed", "ip", ip, "attempt", attempt+1, "error", lastErr)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// fetchIPInfo performs the actual HTTP request
func (c *Client) fetchIPInfo(ctx context.Context, ip string) (*models.ShodanIPInfo, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, ip)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "sdlookup/2.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		c.logger.Warn("rate limited by API", "ip", ip)
		return nil, fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Check for "No information available" response
	bodyStr := string(body)
	if bodyStr == "" || bodyStr == "{}" {
		return &models.ShodanIPInfo{IP: ip}, nil
	}

	var info models.ShodanIPInfo
	if err := jsoniter.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &info, nil
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens    chan struct{}
	ticker    *time.Ticker
	stop      chan struct{}
	mu        sync.Mutex
	closeOnce sync.Once
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}

	interval := time.Minute / time.Duration(requestsPerMinute)
	limiter := &RateLimiter{
		tokens: make(chan struct{}, requestsPerMinute),
		ticker: time.NewTicker(interval),
		stop:   make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < requestsPerMinute; i++ {
		limiter.tokens <- struct{}{}
	}

	// Refill tokens over time
	go func() {
		for {
			select {
			case <-limiter.ticker.C:
				select {
				case limiter.tokens <- struct{}{}:
				default:
					// Buffer full, skip
				}
			case <-limiter.stop:
				return
			}
		}
	}()

	return limiter
}

// Wait blocks until a token is available
func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-r.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the rate limiter
func (r *RateLimiter) Close() {
	r.closeOnce.Do(func() {
		close(r.stop)
		r.ticker.Stop()
	})
}
