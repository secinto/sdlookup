# Bug Fix: omitEmpty Configuration Respected Throughout Application

## Issue Summary

**Bug**: The `omitEmpty` configuration was being ignored during output processing, causing empty scan results to be filtered out even when users explicitly set `omitEmpty=false` to see all results.

## Root Causes

### 1. **Output Processing (main.go)** - FIXED
Both `processInput()` and `processStdin()` had hardcoded checks that always filtered empty results:
```go
// Before (BUG):
if result.Error == nil && result.Info != nil && !result.Info.IsEmpty() {
    // Always filtered if IsEmpty() returns true
}
```

### 2. **Formatter Limitation (formatter.go)** - FIXED
Formatters iterated over port arrays, producing empty strings for empty results:
```go
// Before (BUG):
for _, port := range result.Info.Ports {  // Empty array = no output
    lines = append(lines, fmt.Sprintf("%s:%d", result.IP, port))
}
```

### 3. **Scanner Redundant Logic (scanner.go)** - CLEANED UP
Scanner had misleading code that logged "omitting" but still returned results:
```go
// Before (CONFUSING):
if s.omitEmpty && info.IsEmpty() {
    s.logger.Debug("omitting empty result", "ip", ip)
    return &models.ScanResult{IP: ip, Info: info}, nil  // Still returns!
}
return &models.ScanResult{IP: ip, Info: info}, nil  // Same thing!
```

## Fixes Applied

### 1. Main Processing Logic (cmd/sdlookup/main.go)

**processInput() - Lines 217-224:**
```go
// After (FIXED):
if result.Error == nil && result.Info != nil {
    // Respect cfg.OmitEmpty configuration
    if !cfg.OmitEmpty || !result.Info.IsEmpty() {
        writer.Write(result)
        servicesCollector.Add(result)
    }
}
```

**processStdin() - Lines 278-292:**
```go
// After (FIXED):
hasData := result.Error == nil && result.Info != nil
success := hasData && (!cfg.OmitEmpty || !result.Info.IsEmpty())
progress.Increment(success)

if hasData {
    // Respect cfg.OmitEmpty configuration
    if !cfg.OmitEmpty || !result.Info.IsEmpty() {
        writer.Write(result)
        servicesCollector.Add(result)
    }
}
```

### 2. Formatter Enhancements (internal/output/formatter.go)

**CSV Formatter:**
```go
// If no ports, show IP with metadata (for omitEmpty=false case)
if len(result.Info.Ports) == 0 {
    if f.onlyIPPort {
        return fmt.Sprintf("%s", result.IP), nil
    }
    return fmt.Sprintf("%s,(no ports),%s,%s,%s,%s",
        result.IP, hostnames, tags, cpes, vulns), nil
}
```

**Simple Formatter:**
```go
// If no ports, just show IP (for omitEmpty=false case)
if len(result.Info.Ports) == 0 {
    return result.IP, nil
}
```

### 3. Scanner Cleanup (internal/scanner/scanner.go)

```go
// After (CLEANED UP):
// Log if result is empty (filtering happens at output layer)
if s.omitEmpty && info.IsEmpty() {
    s.logger.Debug("result is empty", "ip", ip)
}

return &models.ScanResult{
    IP:   ip,
    Info: info,
}, nil
```

## Architecture

The fix implements proper **separation of concerns**:

```
Scanner Layer:
  ├─ Always returns all results
  ├─ Logs when result is empty (for debugging)
  └─ Does NOT filter

Output Processing Layer:
  ├─ Checks cfg.OmitEmpty configuration
  ├─ Filters based on user preference
  └─ Handles display logic

Formatter Layer:
  ├─ Handles empty results gracefully
  ├─ Outputs IP even with no ports
  └─ Enables omitEmpty=false to work
```

## Behavior Changes

### Before (Buggy):
```bash
# Private IP with no ports
echo "192.168.1.1" | sdlookup -omitEmpty=false
# Output: (nothing - incorrectly filtered!)
```

### After (Fixed):
```bash
# Simple format
echo "192.168.1.1" | sdlookup -omitEmpty=false -open
# Output: 192.168.1.1

# CSV format
echo "192.168.1.1" | sdlookup -omitEmpty=false
# Output: 192.168.1.1,(no ports),,,,

# JSON format
echo "192.168.1.1" | sdlookup -omitEmpty=false -json
# Output: {"ip":"192.168.1.1","ports":[],"hostnames":[],...}
```

## Tests

### New Tests (cmd/sdlookup/main_test.go)
```go
TestOmitEmptyConfiguration:
  ✓ omitEmpty=true, non-empty result  → Show (expected)
  ✓ omitEmpty=true, empty result      → Hide (expected)
  ✓ omitEmpty=false, non-empty result → Show (expected)
  ✓ omitEmpty=false, empty result     → Show (BUG FIX!)

TestOmitEmptyIntegration:
  ✓ Config merging with flags
```

### Updated Tests (internal/output/formatter_test.go)
```go
TestCSVFormatter_Format:
  ✓ empty result - simple format   → "192.168.1.1"
  ✓ empty result - full CSV format → "192.168.1.1,(no ports),,,,"

TestSimpleFormatter_Format:
  ✓ with ports     → "IP:Port\n..."
  ✓ empty ports    → "192.168.1.1"
```

## Test Results

```
✓ cmd/sdlookup:      2/2 tests passing
✓ internal/api:      6/6 tests passing
✓ internal/config:   4/4 tests passing
✓ internal/models:   5/5 tests passing
✓ internal/output:   6/6 tests passing
✓ pkg/validator:     5/5 tests passing

Total: 28/28 tests PASSING
```

## User Impact

Users can now:
- ✅ Set `omitEmpty=false` in config or via CLI flag
- ✅ See ALL scanned IPs, including those with no open ports
- ✅ Track scan coverage and identify IPs with no exposed services
- ✅ Use for compliance reporting (show all targets, not just findings)
- ✅ Debug scan behavior more effectively

## Use Cases

### 1. Scan Coverage Verification
```bash
# See which IPs were scanned but have no ports
cat network.txt | sdlookup -omitEmpty=false | grep "(no ports)"
```

### 2. Compliance Reporting
```bash
# Show all IPs in report, mark those with no findings
sdlookup -omitEmpty=false 10.0.0.0/24 > full_audit.csv
```

### 3. Debugging
```bash
# Verify scanner reached all IPs in range
echo "192.168.1.0/24" | sdlookup -omitEmpty=false -v | wc -l
# Should show 256 results
```

## Commits

1. **07ea018** - Initial fix for main.go and formatters
2. **[NEXT]** - Cleanup scanner redundant code

## Related Files

- `cmd/sdlookup/main.go` - Output processing logic
- `cmd/sdlookup/main_test.go` - Integration tests
- `internal/scanner/scanner.go` - Scanner cleanup
- `internal/output/formatter.go` - Formatter enhancements
- `internal/output/formatter_test.go` - Formatter tests
