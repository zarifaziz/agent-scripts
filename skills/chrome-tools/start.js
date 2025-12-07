#!/usr/bin/env node
import { spawn, execSync } from "node:child_process";
import puppeteer from "puppeteer-core";
import fs from "node:fs";
import path from "node:path";

const useProfile = process.argv[2] === "--profile";

if (process.argv[2] && process.argv[2] !== "--profile") {
    console.log("Usage: start.js [--profile]");
    console.log("\nOptions:");
    console.log("  --profile  Copy your default Thorium profile (cookies, logins)");
    console.log("\nExamples:");
    console.log("  start.js            # Start with fresh profile");
    console.log("  start.js --profile  # Start with your Thorium profile");
    process.exit(1);
}

// Kill existing Thorium
try {
    execSync("pkill -f 'Thorium'", { stdio: "ignore" });
} catch {}

// Wait a bit for processes to fully die
await new Promise((r) => setTimeout(r, 1000));

// Config matching chrome-flutter-extension.sh
const homeDir = process.env["HOME"];
const thoriumProfilePath = path.join(homeDir, "Library", "Application Support", "Thorium", "Profile 1");
const thoriumUserDataDir = path.join(homeDir, "Library", "Application Support", "Thorium");
const cacheDir = path.join(homeDir, ".cache", "scraping");

// Setup cache directory for fresh profile mode
if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true });
}

let userDataDir = cacheDir;
let profileDir = "";

if (useProfile) {
    // Use existing Thorium profile directly (same as chrome-flutter-extension.sh)
    userDataDir = thoriumUserDataDir;
    profileDir = "Profile 1";
}

// Build args
const chromeArgs = [
    "--remote-debugging-port=9222",
    `--user-data-dir=${userDataDir}`,
];

if (profileDir) {
    chromeArgs.push(`--profile-directory=${profileDir}`);
}

// Start Thorium in background (detached so Node can exit)
spawn(
    "/Applications/Thorium.app/Contents/MacOS/Thorium",
    chromeArgs,
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
    console.error("x Failed to connect to Thorium");
    process.exit(1);
}

console.log(`ok Thorium started on :9222${useProfile ? " with your profile" : ""}`);
