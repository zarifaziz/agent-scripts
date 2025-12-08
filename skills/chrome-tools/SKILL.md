---
name: browser-tools
description: Minimal CDP tools for collaborative site exploration.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Browser Tools

Minimal CDP tools for collaborative site exploration.

## Start Thorium

```bash
start.js              # Fresh profile
start.js --profile    # Use your Thorium Profile 1 (cookies, logins)
```

Start Thorium on `:9222` with remote debugging (uses Profile 1 from chrome-flutter-extension.sh).

## Navigate

```bash
nav.js https://example.com
nav.js https://example.com --new
```

Navigate current tab or open new tab.

## Evaluate JavaScript

```bash
eval.js 'document.title'
eval.js 'document.querySelectorAll("a").length'
```

Execute JavaScript in active tab (async context).

## Screenshot

```bash
screenshot.js                    # Default: 50% scale, quality 50, webp
screenshot.js --scale=1          # Full resolution
screenshot.js --quality=80       # Higher quality (1-100)
screenshot.js --full             # Full page scroll capture
```

Screenshot current viewport, returns temp file path. Optimized for LLM token efficiency.

## Pick Elements

```bash
pick.js "Click the submit button"
```

Interactive element picker. Click to select, Cmd/Ctrl+Click for multi-select, Enter to finish.
