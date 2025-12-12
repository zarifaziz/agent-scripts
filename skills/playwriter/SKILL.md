# Playwriter Skill

Browser automation via Playwright MCP. Uses a single `execute` tool that runs Playwright code.

## Usage

```javascript
// Navigate
await page.goto('https://example.com')

// Click
await page.click('button')

// Type
await page.fill('input[name="search"]', 'query')

// Screenshot
await page.screenshot({ path: 'screenshot.png' })

// Get accessibility snapshot
const snapshot = await accessibilitySnapshot()
console.log(snapshot)
```

## Icon States

- Gray: Not connected
- Green: Connected and ready
- Orange (...): Connecting
- Red (!): Error

Click extension icon on a tab to connect it before automation.
