# Bug Fix: Rate Limiter Goroutine Leak

## Issue Summary

**Bug**: Goroutine leak when `WithRateLimit` replaces the default rate limiter, leaving background goroutines running indefinitely and consuming system resources.

## Root Cause

### Background Goroutine in RateLimiter

The `NewRateLimiter()` function (internal/api/client.go:230-264) starts a background goroutine that runs indefinitely:

```go
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
            return  // Only exits when Close() is called
        }
    }
}()
```

This goroutine refills rate limiter tokens over time and only stops when `Close()` is called on the limiter.

### Leak in WithRateLimit

The `WithRateLimit()` option function (lines 60-68) replaced the rate limiter without cleaning up the old one:

```go
// Before (BUG):
func WithRateLimit(requestsPerMinute int) ClientOption {
    return func(c *Client) {
        c.limiter = NewRateLimiter(requestsPerMinute)  // Creates new goroutine
        // Old limiter's goroutine keeps running!
    }
}
```

### Leak on Client Creation

`NewClient()` creates a default rate limiter (line 112):

```go
limiter: NewRateLimiter(60), // default 60 req/min
```

When `WithRateLimit()` is called, it creates a new limiter and abandons the default one without calling `Close()`, causing:
- The old goroutine continues running
- Memory leak as abandoned limiters pile up
- Potential CPU waste from unnecessary ticker operations

## Fixes Applied

### 1. WithRateLimit Cleanup (internal/api/client.go:60-69)

```go
// After (FIXED):
func WithRateLimit(requestsPerMinute int) ClientOption {
    return func(c *Client) {
        // Close old limiter to stop its background goroutine
        if c.limiter != nil {
            c.limiter.Close()
        }
        c.limiter = NewRateLimiter(requestsPerMinute)
    }
}
```

### 2. Client.Close() Method (internal/api/client.go:284-290)

Added a proper cleanup method to the Client:

```go
// Close cleans up client resources including stopping the rate limiter
func (c *Client) Close() error {
    if c.limiter != nil {
        c.limiter.Close()
    }
    return nil
}
```

### 3. RateLimiter.Close() Safety

The existing `Close()` method already has proper safety with `sync.Once`:

```go
func (r *RateLimiter) Close() {
    r.closeOnce.Do(func() {
        close(r.stop)
        r.ticker.Stop()
    })
}
```

This ensures:
- Multiple calls to `Close()` are safe (idempotent)
- No panic from closing channels multiple times
- Ticker is properly stopped

## Test Coverage

### New Tests (internal/api/client_test.go)

Created comprehensive tests to verify goroutine cleanup:

```go
TestRateLimiterClose:
  ✓ Verifies goroutine count decreases after Close()
  ✓ Confirms cleanup happens properly

TestWithRateLimitClosesOldLimiter:
  ✓ Verifies replacing rate limiter doesn't leak goroutines
  ✓ Confirms old limiter is cleaned up before new one is created

TestClientClose:
  ✓ Verifies Client.Close() properly cleans up rate limiter
  ✓ Confirms goroutine count decreases

TestRateLimiterDoubleClose:
  ✓ Verifies Close() is idempotent
  ✓ No panic on multiple Close() calls

TestRateLimiterWaitAfterClose:
  ✓ Verifies Wait() doesn't block after Close()
  ✓ Prevents deadlock scenarios

TestClientCloseNilLimiter:
  ✓ Verifies Close() handles nil limiter (edge case)
  ✓ No panic on nil dereference
```

### Test Results

```
=== RUN   TestRateLimiterClose
--- PASS: TestRateLimiterClose (0.15s)
=== RUN   TestWithRateLimitClosesOldLimiter
--- PASS: TestWithRateLimitClosesOldLimiter (0.25s)
=== RUN   TestClientClose
--- PASS: TestClientClose (0.15s)
=== RUN   TestRateLimiterDoubleClose
--- PASS: TestRateLimiterDoubleClose (0.00s)
=== RUN   TestRateLimiterWaitAfterClose
--- PASS: TestRateLimiterWaitAfterClose (0.00s)
=== RUN   TestClientCloseNilLimiter
--- PASS: TestClientCloseNilLimiter (0.00s)
PASS
ok      github.com/h4sh5/sdlookup/internal/api  0.723s
```

## Behavior Changes

### Before (Buggy):

```go
// Create client with default 60 req/min limiter
client := api.NewClient(5 * time.Second)  // Goroutine #1 starts

// Change to 120 req/min
api.WithRateLimit(120)(client)  // Goroutine #2 starts, #1 keeps running!

// Change to 200 req/min
api.WithRateLimit(200)(client)  // Goroutine #3 starts, #1 and #2 keep running!

// Result: 3 goroutines running, 2 leaked
```

### After (Fixed):

```go
// Create client with default 60 req/min limiter
client := api.NewClient(5 * time.Second)  // Goroutine #1 starts

// Change to 120 req/min
api.WithRateLimit(120)(client)  // Goroutine #1 stopped, #2 starts

// Change to 200 req/min
api.WithRateLimit(200)(client)  // Goroutine #2 stopped, #3 starts

// Clean up
client.Close()  // Goroutine #3 stopped

// Result: 0 goroutines running, 0 leaked
```

## Impact Assessment

### Severity: High

- **Memory Leak**: Each leaked limiter holds memory for channels, tickers, and goroutine stack
- **Resource Waste**: Leaked goroutines consume CPU cycles on ticker events
- **Scalability Issue**: Long-running applications or frequent config changes accumulate leaks

### User Impact:

- ✅ Applications can now safely change rate limits without leaking goroutines
- ✅ Long-running instances no longer accumulate memory/CPU waste
- ✅ Proper cleanup with `client.Close()` prevents resource leaks

## Usage Recommendations

### 1. Always Close Clients

```go
client := api.NewClient(5 * time.Second)
defer client.Close()  // Ensures cleanup
```

### 2. Safe Rate Limit Changes

```go
// Can safely change rate limits multiple times
api.WithRateLimit(120)(client)
api.WithRateLimit(200)(client)
// Old limiters are properly cleaned up
```

### 3. Configuration Reload

```go
// Safe to reload config with new rate limits
func reloadConfig(client *api.Client, newRate int) {
    api.WithRateLimit(newRate)(client)
    // Old limiter is cleaned up automatically
}
```

## Architecture

The fix maintains proper **resource lifecycle management**:

```
Client Lifecycle:
  ├─ NewClient()
  │   └─ Creates default RateLimiter (goroutine starts)
  │
  ├─ WithRateLimit() [optional, can be called multiple times]
  │   ├─ Closes old limiter (goroutine stops)
  │   └─ Creates new limiter (new goroutine starts)
  │
  └─ Close()
      └─ Closes limiter (goroutine stops)

RateLimiter Lifecycle:
  ├─ NewRateLimiter()
  │   └─ Starts background goroutine
  │
  └─ Close()
      ├─ Closes stop channel (goroutine exits)
      ├─ Stops ticker
      └─ Uses sync.Once for safety
```

## Related Issues

This fix prevents a class of resource leaks that can occur when:
- Configuration is reloaded dynamically
- Rate limits are adjusted based on API responses
- Multiple clients are created and destroyed
- Long-running services accumulate leaked goroutines

## Files Modified

- `internal/api/client.go` - Added Close() method, fixed WithRateLimit
- `internal/api/client_test.go` - Added comprehensive goroutine leak tests

## Verification

To verify no goroutine leaks in your application:

```go
import "runtime"

// Before
initial := runtime.NumGoroutine()

// Create and use client
client := api.NewClient(5 * time.Second)
api.WithRateLimit(120)(client)
client.Close()

// After
time.Sleep(100 * time.Millisecond)
final := runtime.NumGoroutine()

if final >= initial {
    log.Printf("Possible goroutine leak: before=%d after=%d", initial, final)
}
```

## Prevention

To prevent similar issues in the future:
1. Always pair resource creation with cleanup methods
2. Use `defer` for cleanup in goroutines
3. Test goroutine counts in unit tests
4. Use `sync.Once` for idempotent cleanup
5. Document lifecycle management in comments
