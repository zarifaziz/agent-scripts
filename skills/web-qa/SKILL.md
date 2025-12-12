---
name: web-qa
description: QA testing workflow for web applications using browser-tools. Use when testing frontend UI rendering, form inputs, or visual verification.
allowed-tools:
  - bash
  - Read
  - create_file
  - edit_file
metadata:
  version: "1.0"
---

# Web QA Testing Skill

Structured workflow for QA testing web applications. Combines browser-tools with systematic test reporting.

## What Parent Should Provide

When receiving a QA task, expect these in the prompt:

```markdown
## App Overview
[Name] - [What it does]

## Server Commands
- Backend: [command] (port X)
- Frontend: [command] (port Y)

## Valid Inputs & Expected Outputs
- [input example] → [expected result]

## Features to Test (priority order)
1. [Feature]: [trigger] → [expected]

## Testable Functions (if available)
window.__TEST__.functionName()

## Critical Paths (blockers)
1. [Must work or stop]
```

If missing, ask parent or explore with browser-tools.

## Prerequisites

This skill requires `browser-tools` skill to be available. The browser tools are in PATH:
- `browser-start.js` - Launch Chrome with remote debugging
- `browser-nav.js` - Navigate to URLs
- `browser-screenshot.js` - Capture screenshots
- `browser-eval.js` - Run JS in page context

## Standard QA Workflow

### 1. Kill Existing Servers First

**ALWAYS** kill existing servers before starting new ones to avoid port conflicts:

```bash
pkill -f "node.*server" 2>/dev/null
pkill -f "vite" 2>/dev/null
sleep 1
```

### 2. Start Servers and Verify

Start backend and frontend separately, verify each is running:

```bash
# Backend - check logs for actual port
cd $PROJECT_ROOT && npm run server > /tmp/server.log 2>&1 &
sleep 2
cat /tmp/server.log  # Find actual port

# Frontend
cd $PROJECT_ROOT/web && npm run dev > /tmp/frontend.log 2>&1 &
sleep 5
lsof -i :3000 | head -3  # Verify listening
```

**IMPORTANT**: Don't assume ports! Read server output to find actual ports.

### 3. Start Browser and Navigate

```bash
browser-start.js
browser-nav.js http://localhost:3000
sleep 2  # Wait for React/Vue to hydrate
```

### 4. Screenshot and Verify UI

```bash
browser-screenshot.js
# Returns temp path like /var/folders/.../screenshot-xxx.png
# Use Read tool to view the image
```

### 5. Test DOM Elements

Use IIFE pattern for browser-eval.js (avoids syntax errors with const/let):

```bash
# Check element exists
browser-eval.js 'document.querySelector("h1")?.textContent || "Not found"'

# Test input works - MUST use IIFE pattern
browser-eval.js '(function() { var el = document.querySelector("textarea"); if (el) { el.value = "test"; return "OK"; } return "No input"; })()'
```

**GOTCHA**: browser-eval.js doesn't support top-level const/let. Wrap in IIFE or use var.

### 6. Save Screenshots to Project

Always save to project's `qa-screenshots/` folder:

```bash
mkdir -p $PROJECT_ROOT/qa-screenshots
cp /var/folders/.../screenshot-xxx.png $PROJECT_ROOT/qa-screenshots/descriptive-name.png
```

Use descriptive names: `initial-load.png`, `after-input.png`, `error-state.png`

### 7. Update QA.md Report

Create/update `QA.md` in project root with structured results:

```markdown
# QA Test Results - [Project Name]

**Date:** YYYY-MM-DD
**Tester:** Automated QA Agent

## Screenshots
Located in `/qa-screenshots/`:
- `screenshot-name.png` - Description

## Test Results

| Component | Status | Notes |
|-----------|--------|-------|
| Component Name | PASS/FAIL | Details |

## Issues Found
[List any bugs or blockers]

## Overall Assessment
**PASS/FAIL** - Summary
```

## Checklist for UI Testing

Standard components to verify:

- [ ] Page loads without blank screen
- [ ] Title/header renders
- [ ] Main content area visible
- [ ] Input fields present and focusable
- [ ] Can type in inputs
- [ ] Buttons visible and labeled
- [ ] Theme/styling applied
- [ ] No console errors (check with browser-eval.js)

## Common Failures and Fixes

### Blank White Page
- React not mounting - check console for errors
- Check if `useLocalRuntime` vs `useLocalThreadRuntime` (assistant-ui)
- Build succeeded but runtime error

### Dark Background Only (No UI)
- Components fail to render
- Check for hook errors
- Verify all providers wrap the app

### Port Already in Use
- Always kill servers first
- Check `lsof -i :PORT` before starting

## Example Console Error Check

```bash
browser-eval.js '(function() { var errors = []; window.onerror = function(m) { errors.push(m); }; return JSON.stringify(errors); })()'
```

## App Test Hooks (for Developers)

If you control the app, export test hooks for stable QA:

```typescript
// Add to app entry point (dev mode only)
if (import.meta.env.DEV) {
  window.__TEST__ = {
    getState: () => store.getState(),
    getInputValue: () => document.querySelector('textarea')?.value,
    submitInput: (text) => { /* trigger form submit */ },
    isLoading: () => runtime.isRunning,
  }
}
```

Then QA uses stable interface instead of fragile DOM selectors:
```bash
# Stable
browser-eval.js 'window.__TEST__.getInputValue()'

# Fragile - breaks on refactor
browser-eval.js 'document.querySelector(".aui-composer textarea")?.value'
```

**Note**: For parent sessions spawning QA servants, see `spawn-servant` skill for how to write QA contracts.

## Quick Reference

```bash
# Full QA flow in one go
pkill -f "node.*server"; pkill -f "vite"; sleep 1
cd $PROJECT && npm run server > /tmp/s.log 2>&1 &
cd $PROJECT/web && npm run dev > /tmp/f.log 2>&1 &
sleep 5
browser-start.js
browser-nav.js http://localhost:3000
sleep 2
browser-screenshot.js  # then Read the path
```
