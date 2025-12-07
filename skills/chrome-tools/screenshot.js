#!/usr/bin/env node
import { tmpdir } from "node:os";
import { join } from "node:path";
import puppeteer from "puppeteer-core";

// Parse args: --quality=N (1-100, default 50), --full (full page), --scale=N (0.1-1, default 0.5)
const args = process.argv.slice(2);
const qualityArg = args.find(a => a.startsWith('--quality='));
const scaleArg = args.find(a => a.startsWith('--scale='));
const quality = qualityArg ? parseInt(qualityArg.split('=')[1]) : 50;
const scale = scaleArg ? parseFloat(scaleArg.split('=')[1]) : 0.5;
const fullPage = args.includes('--full');

const b = await puppeteer.connect({
    browserURL: "http://localhost:9222",
    defaultViewport: null,
});

const pages = await b.pages();
const p = pages[pages.length - 1];

if (!p) {
    console.error("x No active tab found");
    process.exit(1);
}

// Get current viewport and set scaled version
const viewport = await p.viewport();
if (viewport && scale !== 1) {
    await p.setViewport({
        width: Math.round(viewport.width * scale),
        height: Math.round(viewport.height * scale),
        deviceScaleFactor: 1,
    });
}

const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
const filename = `screenshot-${timestamp}.webp`;
const filepath = join(tmpdir(), filename);

await p.screenshot({ 
    path: filepath,
    type: 'webp',
    quality: quality,
    fullPage: fullPage,
});

// Restore viewport
if (viewport && scale !== 1) {
    await p.setViewport(viewport);
}

console.log(filepath);

await b.disconnect();
