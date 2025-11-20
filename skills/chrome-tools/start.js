#!/usr/bin/env node
import { spawn, execSync } from "node:child_process";
import puppeteer from "puppeteer-core";
import fs from "node:fs";
import path from "node:path";

const useProfile = process.argv[2] === "--profile";

if (process.argv[2] && process.argv[2] !== "--profile") {
    console.log("Usage: start.js [--profile]");
    console.log("\nOptions:");
    console.log("  --profile  Copy your default Chrome profile (cookies, logins)");
    console.log("\nExamples:");
    console.log("  start.js            # Start with fresh profile");
    console.log("  start.js --profile  # Start with your Chrome profile");
    process.exit(1);
}

// Kill existing Chrome
try {
    execSync("killall 'Google Chrome'", { stdio: "ignore" });
} catch {}

// Wait a bit for processes to fully die
await new Promise((r) => setTimeout(r, 1000));

// Setup profile directory
const homeDir = process.env["HOME"];
const cacheDir = path.join(homeDir, ".cache", "scraping");
if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
}

if (useProfile) {
    // Sync profile with rsync (much faster on subsequent runs)
    // Note: Ensure the source path matches your system's Chrome profile path
    const sourceProfile = path.join(homeDir, "Library", "Application Support", "Google", "Chrome");
    execSync(
        `rsync -a --delete "${sourceProfile}/" "${cacheDir}/"`,
        { stdio: "pipe" },
    );
}

// Start Chrome in background (detached so Node can exit)
spawn(
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    ["--remote-debugging-port=9222", `--user-data-dir=${cacheDir}`],
    { detached: true, stdio: "ignore" },
).unref();

// Wait for Chrome to be ready by attempting to connect
let connected = false;
for (let i = 0; i < 30; i++) {
    try {
        const browser = await puppeteer.connect({
            browserURL: "http://localhost:9222",
            defaultViewport: null,
        });
        await browser.disconnect();
        connected = true;
        break;
    } catch {
        await new Promise((r) => setTimeout(r, 500));
    }
}

if (!connected) {
    console.error("✗ Failed to connect to Chrome");
    process.exit(1);
}

console.log(`✓ Chrome started on :9222 ${useProfile ? " with your profile" : ""} `);
