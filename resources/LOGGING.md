# Error Handling & Logging Patterns

This guide documents the established error handling patterns in the Resources service.

## The Established Patterns

### ✅ Pattern 1: Log + Return oops (for operational errors)
```go
log.From(ctx).Error("failed to X", 
    zap.String("skillID", string(seed.SkillID)),
    logf.Error(err))
return nil, oops.Wrapf(err, "failed to X")  // simpler message
```
**Use when:** External service calls, DB operations, critical failures needing debugging

### ✅ Pattern 2: Just return oops (for validation errors)
```go
// NO logging - just return
return 0, oops.With("interval", interval).New("invalid number in interval")
```
**Use when:** Validation/format errors, the error message is self-explanatory

### ✅ Pattern 3: Just log, don't return (for non-critical failures)
```go
if err := invalidateSectionNodesCache(r.RedisClient)(ctx, []string{sectionID}); err != nil {
    log.From(ctx).Error("could not evict section nodes cache", logf.Error(err))
}
return node, nil  // continue despite error
```
**Use when:** Cache operations, optional enrichment, graceful degradation

---

## Anti-Pattern: Duplicating Context

```go
// ❌ REDUNDANT - duplicating context in both places
log.From(ctx).Error("spanning API returned question with nil difficulty",
    zap.String("skillID", string(seed.SkillID)),
    zap.String("questionID", string(q.ID)),
    ...
)
return WorksheetSpanningResult{}, oops.
    With("skillID", seed.SkillID).    // duplicated!
    With("questionID", q.ID).          // duplicated!
    ...
    Errorf("spanning API returned question with nil difficulty")
```

## How to Fix It

**Option A:** This is a validation error that shouldn't happen - just use oops with context:
```go
if q.Difficulty == nil {
    return WorksheetSpanningResult{}, oops.
        With("skillID", seed.SkillID).
        With("questionID", q.ID).
        With("questionIndex", q.Index).
        With("aiFallback", q.AIFallback).
        Errorf("spanning API returned question with nil difficulty")
}
```

**Option B:** If you want logging for observability, log it but simplify:
```go
if q.Difficulty == nil {
    log.From(ctx).Error("spanning API returned question with nil difficulty",
        zap.String("skillID", string(seed.SkillID)),
        zap.String("questionID", string(q.ID)),
        zap.Int("questionIndex", q.Index),
        zap.Bool("aiFallback", q.AIFallback),
    )
    return WorksheetSpanningResult{}, oops.Errorf("spanning API returned question with nil difficulty")
}
```

The `oops` library automatically captures stack traces and the `.With()` context gets preserved up the error chain - so the detail is already there for error handlers. The logging is only needed if you want it to appear in logs *even when the error is handled gracefully upstream*.

---
