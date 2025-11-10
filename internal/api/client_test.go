package api

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestRateLimiterClose(t *testing.T) {
	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create a rate limiter
	limiter := NewRateLimiter(60)

	// Wait a bit for goroutine to start
	time.Sleep(50 * time.Millisecond)

	// Verify goroutine was created
	afterCreate := runtime.NumGoroutine()
	if afterCreate <= initialGoroutines {
		t.Errorf("Expected goroutine count to increase after creating rate limiter, got %d (initial: %d)",
			afterCreate, initialGoroutines)
	}

	// Close the limiter
	limiter.Close()

	// Wait for goroutine to stop
	time.Sleep(100 * time.Millisecond)

	// Verify goroutine was cleaned up
	afterClose := runtime.NumGoroutine()
	if afterClose >= afterCreate {
		t.Errorf("Expected goroutine count to decrease after closing rate limiter, got %d (after create: %d)",
			afterClose, afterCreate)
	}
}

func TestWithRateLimitClosesOldLimiter(t *testing.T) {
	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create client with default rate limiter
	client := NewClient(5 * time.Second)

	// Wait for initial goroutine to start
	time.Sleep(50 * time.Millisecond)

	afterFirst := runtime.NumGoroutine()
	if afterFirst <= initialGoroutines {
		t.Errorf("Expected goroutine count to increase after creating client, got %d (initial: %d)",
			afterFirst, initialGoroutines)
	}

	// Replace rate limiter using WithRateLimit
	WithRateLimit(120)(client)

	// Wait for old goroutine to stop and new one to start
	time.Sleep(100 * time.Millisecond)

	afterReplace := runtime.NumGoroutine()

	// Should have approximately the same number of goroutines (one old stopped, one new started)
	// Allow some tolerance for test flakiness
	diff := afterReplace - afterFirst
	if diff > 1 {
		t.Errorf("Expected goroutine count to remain stable after replacing rate limiter, "+
			"but got %d extra goroutines (before: %d, after: %d)",
			diff, afterFirst, afterReplace)
	}

	// Clean up
	client.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestClientClose(t *testing.T) {
	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create client
	client := NewClient(5 * time.Second)

	// Wait for goroutine to start
	time.Sleep(50 * time.Millisecond)

	afterCreate := runtime.NumGoroutine()
	if afterCreate <= initialGoroutines {
		t.Errorf("Expected goroutine count to increase after creating client, got %d (initial: %d)",
			afterCreate, initialGoroutines)
	}

	// Close client
	err := client.Close()
	if err != nil {
		t.Fatalf("Client.Close() returned error: %v", err)
	}

	// Wait for goroutine to stop
	time.Sleep(100 * time.Millisecond)

	// Verify goroutine was cleaned up
	afterClose := runtime.NumGoroutine()
	if afterClose >= afterCreate {
		t.Errorf("Expected goroutine count to decrease after closing client, got %d (after create: %d)",
			afterClose, afterCreate)
	}
}

func TestRateLimiterDoubleClose(t *testing.T) {
	// Create a rate limiter
	limiter := NewRateLimiter(60)

	// Close twice - should not panic
	limiter.Close()
	limiter.Close()

	// Test passed if no panic occurred
}

func TestRateLimiterWaitAfterClose(t *testing.T) {
	// Create a rate limiter
	limiter := NewRateLimiter(60)

	// Close it
	limiter.Close()

	// Try to wait - should return immediately with context.Canceled or nil
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should not block indefinitely
	done := make(chan struct{})
	go func() {
		limiter.Wait(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success - wait returned
	case <-time.After(2 * time.Second):
		t.Error("Wait() blocked after Close() - should have returned immediately")
	}
}

func TestClientCloseNilLimiter(t *testing.T) {
	// Create a client and set limiter to nil (edge case)
	client := &Client{
		limiter: nil,
	}

	// Should not panic
	err := client.Close()
	if err != nil {
		t.Fatalf("Client.Close() with nil limiter returned error: %v", err)
	}
}
