---
name: browser-tools
description: Chrome DevTools Protocol tools for agent-assisted web automation.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Browser Tools

Chrome DevTools Protocol tools for agent-assisted web automation. These tools connect to Chrome running on `:9222` with remote debugging enabled.

## Install

Assume already installed and only run the below install if existing commands fail

```bash
pnpm install
```

## How to Invoke These Tools

## Start Chrome

```bash
browser-start.js              # Fresh profile
browser-start.js --profile    # Copy user's profile (cookies, logins)
```

Launch Chrome with remote debugging on `:9222`. Use `--profile` to preserve user's authentication state.

## Navigate

```bash
browser-nav.js https://example.com
browser-nav.js https://example.com --new
```

Navigate to URLs. Use `--new` flag to open in a new tab instead of reusing current tab.

## Console Logs

```bash
browser-console.js
```

Read all logs captured since the last `browser-nav.js` navigation. Includes both JavaScript console calls (log, warn, error, info) AND browser-level errors (CORS violations, network failures, security warnings). No reload needed - logs are buffered from navigation time.

## Evaluate JavaScript

```bash
browser-eval.js 'document.title'
browser-eval.js 'document.querySelectorAll("a").length'
```

Execute JavaScript in the active tab. Code runs in async context. Use this to extract data, inspect page state, or perform DOM operations programmatically.

## Screenshot

```bash
browser-screenshot.js
```

Capture current viewport and return temporary file path. Use this to visually inspect page state or verify UI changes.

## Pick Elements

```bash
browser-pick.js "Click the submit button"
```

**IMPORTANT**: Use this tool when the user wants to select specific DOM elements on the page. This launches an interactive picker that lets the user click elements to select them. The user can select multiple elements (Cmd/Ctrl+Click) and press Enter when done. The tool returns CSS selectors for the selected elements.

Common use cases:

- User says "I want to click that button" → Use this tool to let them select it
- User says "extract data from these items" → Use this tool to let them select the elements
- When you need specific selectors but the page structure is complex or ambiguous

## Cookies

```bash
browser-cookies.js
```

Display all cookies for the current tab including domain, path, httpOnly, and secure flags. Use this to debug authentication issues or inspect session state.

## Extract Page Content

```bash
browser-content.js https://example.com
```

Navigate to a URL and extract readable content as markdown. Uses Mozilla Readability for article extraction and Turndown for HTML-to-markdown conversion. Works on pages with JavaScript content (waits for page to load).
