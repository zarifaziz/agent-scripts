#!/usr/bin/env node

import puppeteer from "puppeteer-core";
import { writeFileSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";

const url = process.argv[2];
const newTab = process.argv[3] === "--new";

if (!url) {
	console.log("Usage: browser-nav.js <url> [--new]");
	console.log("\nExamples:");
	console.log("  browser-nav.js https://example.com       # Navigate current tab");
	console.log("  browser-nav.js https://example.com --new # Open in new tab");
	process.exit(1);
}

const b = await Promise.race([
	puppeteer.connect({
		browserURL: "http://localhost:9222",
		defaultViewport: null,
	}),
	new Promise((_, reject) => setTimeout(() => reject(new Error("timeout")), 5000)),
]).catch((e) => {
	console.error("✗ Could not connect to browser:", e.message);
	console.error("  Run: browser-start.js");
	process.exit(1);
});

// Console capture script to inject before page loads
const consoleCapture = `
window.__consoleLogs = window.__consoleLogs || [];
window.__consoleOriginal = window.__consoleOriginal || {};
['log', 'warn', 'error', 'info', 'debug'].forEach(level => {
	if (!window.__consoleOriginal[level]) {
		window.__consoleOriginal[level] = console[level].bind(console);
		console[level] = (...args) => {
			window.__consoleLogs.push({
				level,
				message: args.map(a => {
					if (typeof a === 'object') {
						try { return JSON.stringify(a); } catch { return String(a); }
					}
					return String(a);
				}).join(' '),
				timestamp: Date.now(),
				location: new Error().stack?.split('\\n')[2]?.trim() || ''
			});
			window.__consoleOriginal[level](...args);
		};
	}
});
`;

let p;
if (newTab) {
	p = await b.newPage();
} else {
	const allPages = await b.pages();
	const pages = allPages.filter((page) => {
		const u = page.url();
		return !u.startsWith("devtools://") && !u.startsWith("chrome-extension://") && !u.startsWith("chrome://");
	});
	p = pages.at(-1) || (await b.newPage());
}

// Collect browser-level logs via CDP
const browserLogs = [];
const client = await p.createCDPSession();
await client.send("Log.enable");
await client.send("Runtime.enable");

// Capture browser log entries (CORS errors, network failures, security warnings)
client.on("Log.entryAdded", (event) => {
	const entry = event.entry;
	browserLogs.push({
		level: entry.level,
		message: entry.text,
		location: entry.url ? `${entry.url.split("/").pop()}:${entry.lineNumber || 1}` : "",
		timestamp: Date.now(),
		source: "browser",
	});
});

// Capture uncaught exceptions
client.on("Runtime.exceptionThrown", (event) => {
	const ex = event.exceptionDetails;
	const text = ex.exception?.description || ex.text || "Unknown error";
	browserLogs.push({
		level: "error",
		message: text,
		location: ex.url ? `${ex.url.split("/").pop()}:${ex.lineNumber || 1}` : "",
		timestamp: Date.now(),
		source: "exception",
	});
});

// Inject console capture before any page scripts run
await p.evaluateOnNewDocument(consoleCapture);

await p.goto(url, { waitUntil: "domcontentloaded" });

// Wait a bit for async errors to come in
await new Promise((r) => setTimeout(r, 500));

// Save browser logs to temp file for browser-console.js to read
const logFile = join(tmpdir(), "browser-logs.json");
writeFileSync(logFile, JSON.stringify(browserLogs, null, 2));

console.log(newTab ? "✓ Opened:" : "✓ Navigated to:", url);

await b.disconnect();
