#!/usr/bin/env node

import puppeteer from "puppeteer-core";
import { readFileSync, existsSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";

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

const allPages = await b.pages();
const pages = allPages.filter((page) => {
	const url = page.url();
	return !url.startsWith("devtools://") && !url.startsWith("chrome-extension://") && !url.startsWith("chrome://");
});

const p = pages.at(-1);

if (!p) {
	console.error("✗ No active tab found");
	process.exit(1);
}

console.log(`Page: ${p.url()}\n`);

// Read console logs captured via monkey-patching
const consoleLogs = await p.evaluate(() => {
	return (window.__consoleLogs || []).map((l) => ({ ...l, source: "console" }));
});

// Read browser-level logs from temp file (CORS, network errors, etc.)
const logFile = join(tmpdir(), "browser-logs.json");
let browserLogs = [];
if (existsSync(logFile)) {
	try {
		browserLogs = JSON.parse(readFileSync(logFile, "utf-8"));
	} catch {
		// ignore parse errors
	}
}

// Merge and sort by timestamp
const allLogs = [...browserLogs, ...consoleLogs].sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0));

// Print logs with color-coded levels
const levelColors = {
	error: "\x1b[31m",
	warn: "\x1b[33m",
	warning: "\x1b[33m",
	info: "\x1b[36m",
	log: "\x1b[37m",
	debug: "\x1b[90m",
	verbose: "\x1b[90m",
};
const reset = "\x1b[0m";

console.log("=".repeat(80));
console.log("CONSOLE OUTPUT");
console.log("=".repeat(80));

if (allLogs.length === 0) {
	console.log("(no console output captured - navigate with browser-nav.js first)");
} else {
	for (const log of allLogs) {
		const color = levelColors[log.level] || "\x1b[37m";
		const levelTag = `[${log.level.toUpperCase()}]`.padEnd(10);
		// Extract file:line from location
		let loc = "";
		if (log.location) {
			const locMatch = log.location.match(/\((.+)\)$/) || log.location.match(/at\s+(.+)$/);
			loc = locMatch ? ` (${locMatch[1].split("/").pop()})` : ` (${log.location})`;
		}
		console.log(`${color}${levelTag}${reset} ${log.message}${loc}`);
	}
}

console.log("=".repeat(80));
console.log(`Total: ${allLogs.length} entries (${browserLogs.length} browser, ${consoleLogs.length} console)`);

await b.disconnect();
