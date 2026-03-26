# eino-ext claude.go Vendor Fix Debug Document

**Date**: 2026-03-26
**Issue**: Panic: index out of range [-1] at claude.go:598
**GitHub Issue**: cloudwego/eino-ext#749

## Problem

When using Claude provider with Kimi API compatibility, a panic occurs when:
- `msgParam.Content` is empty (length 0)
- Code tries to access `msgParam.Content[len(msgParam.Content)-1]` which is `Content[-1]`

## Root Cause

Original code at line ~598:
```go
if ctrl := msgParam.Content[len(msgParam.Content)-1].GetCacheControl(); ctrl != nil && ctrl.Type != "" {
    hasSetMsgBreakPoint = true
}
```

When `msgParam.Content` is an empty slice, `len(msgParam.Content)-1` equals `-1`, causing index out of range panic.

## Fix Applied

**File**: `infrastructure/llm/vendor/github.com/cloudwego/eino-ext/components/model/claude/claude.go`

### Fix 1 (lines 598-603)
```go
// Fix: check Content length before accessing to avoid index out of range panic
if len(msgParam.Content) > 0 {
    if ctrl := msgParam.Content[len(msgParam.Content)-1].GetCacheControl(); ctrl != nil && ctrl.Type != "" {
        hasSetMsgBreakPoint = true
    }
}
```

### Fix 2 (lines 608-614)
```go
if !hasSetMsgBreakPoint && specOptions.AutoCacheControl != nil {
    lastMsgParam := msgParams[len(msgParams)-1]
    // Fix: check Content length before accessing to avoid index out of range panic
    if len(lastMsgParam.Content) > 0 {
        lastBlock := lastMsgParam.Content[len(lastMsgParam.Content)-1]
        populateContentBlockBreakPoint(lastBlock, specOptions.AutoCacheControl)
    }
}
```

## Context

This vendor fix was created as a workaround while waiting for upstream fix in cloudwego/eino-ext#749.

## How to Re-apply

If needed in the future:

1. Locate the eino-ext package in `go.mod` or vendor directory
2. Apply the two fixes above at the corresponding locations in `claude.go`
3. Enable replace directive in `go.mod`:
   ```go
   replace github.com/cloudwego/eino-ext/components/model/claude => ./infrastructure/llm/vendor/github.com/cloudwego/eino-ext/components/model/claude
   ```
4. Run `go mod tidy`

## Notes

- This fix was removed because the vendor directory approach complicates dependency management
- The upstream issue #749 should be tracked for proper fix in eino-ext
- Original issue occurred when LLM returns empty content in tool call responses (Kimi API compatibility)
