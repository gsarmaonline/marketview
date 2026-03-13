#!/usr/bin/env node
/**
 * Screenshot framework for MarketView
 *
 * Captures screenshots of all frontend pages at multiple viewports.
 * Manages the server lifecycle — works with an already-running server,
 * or can start one via docker-compose or the Next.js dev server.
 *
 * Usage:
 *   node scripts/screenshot.js                    # auto-detect server
 *   BASE_URL=http://localhost:3001 npm run screenshot
 *   START_SERVER=docker npm run screenshot         # start via docker-compose
 *   START_SERVER=dev npm run screenshot            # start via next dev
 *   STOP_SERVER=1 npm run screenshot               # stop docker after done
 */

'use strict';

const { chromium } = require('playwright');
const { spawn, execSync } = require('child_process');
const http = require('http');
const https = require('https');
const path = require('path');
const fs = require('fs');

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const config = {
  baseUrl: process.env.BASE_URL || 'http://localhost:3001',

  outputDir: path.resolve(__dirname, '..', 'screenshots'),

  pages: [
    { path: '/', name: 'dashboard', waitFor: '.indicator-card, main' },
    { path: '/portfolio', name: 'portfolio', waitFor: 'main' },
    { path: '/stock/RELIANCE', name: 'stock-reliance', waitFor: 'main', waitUntil: 'load' },
    { path: '/backtest', name: 'backtest', waitFor: 'main' },
  ],

  viewports: [
    { name: 'desktop', width: 1440, height: 900 },
    { name: 'tablet',  width: 768,  height: 1024 },
    { name: 'mobile',  width: 390,  height: 844 },
  ],

  // How long to wait for the server to become ready (ms)
  serverReadyTimeout: 60_000,
  serverPollInterval: 2_000,

  // How long to wait for a page to load (ms)
  pageTimeout: 30_000,
};

// ---------------------------------------------------------------------------
// Server lifecycle helpers
// ---------------------------------------------------------------------------

function probe(url) {
  return new Promise((resolve) => {
    const lib = url.startsWith('https') ? https : http;
    const req = lib.get(url, (res) => {
      res.resume(); // drain
      resolve(res.statusCode < 500);
    });
    req.setTimeout(3_000, () => { req.destroy(); resolve(false); });
    req.on('error', () => resolve(false));
  });
}

async function waitForServer(url, timeout, interval) {
  const deadline = Date.now() + timeout;
  let attempt = 0;
  while (Date.now() < deadline) {
    if (await probe(url)) return true;
    attempt++;
    process.stdout.write(`\r  waiting for ${url} ... (${attempt})`);
    await new Promise((r) => setTimeout(r, interval));
  }
  process.stdout.write('\n');
  return false;
}

function dockerAvailable() {
  try {
    execSync('docker info --format "{{.ServerVersion}}"', { stdio: 'pipe' });
    return true;
  } catch {
    return false;
  }
}

function startDockerCompose() {
  const root = path.resolve(__dirname, '..', '..');
  console.log('  starting docker-compose stack...');
  execSync('docker compose up -d', { cwd: root, stdio: 'inherit' });
}

function stopDockerCompose() {
  const root = path.resolve(__dirname, '..', '..');
  console.log('  stopping docker-compose stack...');
  execSync('docker compose down', { cwd: root, stdio: 'inherit' });
}

function startDevServer() {
  const root = path.resolve(__dirname, '..');
  console.log('  spawning Next.js dev server (npm run dev)...');
  const proc = spawn('npm', ['run', 'dev'], {
    cwd: root,
    stdio: process.env.VERBOSE ? 'inherit' : 'pipe',
    env: { ...process.env },
    detached: false,
  });
  proc.on('error', (err) => {
    console.error('  dev server error:', err.message);
  });
  return proc;
}

/**
 * Ensures the server is reachable.
 * Returns a cleanup function (no-op if server was already running).
 */
async function ensureServer() {
  const { baseUrl } = config;
  const mode = process.env.START_SERVER; // 'docker' | 'dev' | undefined

  // Check if already running
  if (await probe(baseUrl)) {
    console.log(`  server already running at ${baseUrl}`);
    return () => {};
  }

  if (mode === 'docker' || (!mode && dockerAvailable())) {
    startDockerCompose();
    const ready = await waitForServer(baseUrl, config.serverReadyTimeout, config.serverPollInterval);
    if (!ready) throw new Error(`docker-compose stack did not become ready at ${baseUrl}`);
    console.log('\n  stack is ready.');
    return () => {
      if (process.env.STOP_SERVER === '1') stopDockerCompose();
    };
  }

  if (mode === 'dev' || !mode) {
    // Use the dev server URL if docker-compose isn't being used
    if (!process.env.BASE_URL) config.baseUrl = 'http://localhost:3000';
    const devUrl = config.baseUrl;
    const proc = startDevServer();
    const ready = await waitForServer(devUrl, config.serverReadyTimeout, config.serverPollInterval);
    if (!ready) {
      proc.kill();
      throw new Error(`Next.js dev server did not become ready at ${devUrl}`);
    }
    console.log('\n  dev server is ready.');
    return () => {
      console.log('  stopping dev server...');
      proc.kill('SIGTERM');
    };
  }

  throw new Error(
    `Server is not running at ${baseUrl}.\n` +
    `Set START_SERVER=docker or START_SERVER=dev to start one automatically.`
  );
}

// ---------------------------------------------------------------------------
// Screenshot capture
// ---------------------------------------------------------------------------

async function captureAll() {
  const { baseUrl, outputDir, pages, viewports, pageTimeout } = config;

  fs.mkdirSync(outputDir, { recursive: true });

  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const results = [];

  try {
    for (const vp of viewports) {
      const context = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
        // Disable animations for consistent screenshots
        reducedMotion: 'reduce',
      });

      for (const pg of pages) {
        const url = `${baseUrl}${pg.path}`;
        const filename = `${pg.name}-${vp.name}.png`;
        const filepath = path.join(outputDir, filename);

        console.log(`  ${pg.name}  [${vp.name} ${vp.width}x${vp.height}]`);

        const page = await context.newPage();
        try {
          await page.goto(url, { waitUntil: pg.waitUntil || 'networkidle', timeout: pageTimeout });

          // Wait for a representative element to be visible
          if (pg.waitFor) {
            await page.waitForSelector(pg.waitFor, { timeout: pageTimeout }).catch(() => {});
          }

          // Let the UI settle (animations, data loading)
          await page.waitForTimeout(500);

          await page.screenshot({ path: filepath, fullPage: true });
          results.push({ page: pg.name, viewport: vp.name, file: filepath });
        } finally {
          await page.close();
        }
      }

      await context.close();
    }
  } finally {
    await browser.close();
  }

  return results;
}

// ---------------------------------------------------------------------------
// Portfolio seeding helpers
// ---------------------------------------------------------------------------

const RANDOM_STOCKS = [
  { name: 'RELIANCE',  qty: 10,  buy_price: 2400, notes: 'Reliance Industries' },
  { name: 'TCS',       qty: 5,   buy_price: 3800, notes: 'Tata Consultancy Services' },
  { name: 'HDFCBANK',  qty: 20,  buy_price: 1650, notes: 'HDFC Bank' },
  { name: 'INFY',      qty: 15,  buy_price: 1450, notes: 'Infosys' },
  { name: 'WIPRO',     qty: 30,  buy_price: 480,  notes: 'Wipro Ltd' },
  { name: 'ICICIBANK', qty: 25,  buy_price: 1100, notes: 'ICICI Bank' },
  { name: 'BHARTIARTL',qty: 8,   buy_price: 1700, notes: 'Bharti Airtel' },
];

/**
 * Seeds the portfolio with a fixed set of random stock holdings.
 * Returns the list of created holding IDs so they can be cleaned up.
 */
async function seedPortfolio(baseUrl) {
  const ids = [];
  for (const stock of RANDOM_STOCKS) {
    const body = {
      asset_type: 'stock',
      name: stock.name,
      quantity: stock.qty,
      buy_price: stock.buy_price,
      current_value: parseFloat((stock.buy_price * stock.qty * (0.9 + Math.random() * 0.25)).toFixed(2)),
      buy_date: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString(),
      notes: stock.notes,
      metadata: {},
    };
    const res = await fetch(`${baseUrl}/api/portfolio/holdings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      console.warn(`  warning: failed to add ${stock.name}: ${res.status}`);
      continue;
    }
    const data = await res.json();
    if (data.id) ids.push(data.id);
  }
  return ids;
}

/**
 * Deletes holdings by ID to restore a clean portfolio state.
 */
async function cleanupPortfolio(baseUrl, ids) {
  for (const id of ids) {
    await fetch(`${baseUrl}/api/portfolio/holdings/${id}`, { method: 'DELETE' }).catch(() => {});
  }
}

/**
 * Captures portfolio-filled screenshots: seeds holdings, takes screenshots, cleans up.
 */
async function capturePortfolioFilled() {
  const { baseUrl, outputDir, viewports, pageTimeout } = config;

  fs.mkdirSync(outputDir, { recursive: true });

  console.log('  seeding portfolio with random stocks...');
  const seededIds = await seedPortfolio(baseUrl);
  console.log(`  added ${seededIds.length} holdings (ids: ${seededIds.join(', ')})`);

  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const results = [];

  try {
    for (const vp of viewports) {
      const context = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
        reducedMotion: 'reduce',
      });

      const url = `${baseUrl}/portfolio`;
      const filename = `portfolio-filled-${vp.name}.png`;
      const filepath = path.join(outputDir, filename);

      console.log(`  portfolio-filled  [${vp.name} ${vp.width}x${vp.height}]`);

      const page = await context.newPage();
      try {
        await page.goto(url, { waitUntil: 'networkidle', timeout: pageTimeout });
        await page.waitForSelector('.summary-grid, main', { timeout: pageTimeout }).catch(() => {});
        // Wait for table rows to appear
        await page.waitForSelector('table tbody tr', { timeout: 5000 }).catch(() => {});
        await page.waitForTimeout(600);

        await page.screenshot({ path: filepath, fullPage: true });
        results.push({ page: 'portfolio-filled', viewport: vp.name, file: filepath });
      } finally {
        await page.close();
      }

      await context.close();
    }
  } finally {
    await browser.close();
  }

  console.log('  cleaning up seeded holdings...');
  await cleanupPortfolio(baseUrl, seededIds);

  return results;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
  console.log('MarketView screenshot framework');
  console.log(`  base URL : ${config.baseUrl}`);
  console.log(`  output   : ${config.outputDir}`);
  console.log(`  pages    : ${config.pages.map((p) => p.name).join(', ')}, portfolio-filled`);
  console.log(`  viewports: ${config.viewports.map((v) => v.name).join(', ')}`);
  console.log('');

  console.log('[ 1/4 ] checking server...');
  const cleanup = await ensureServer();

  console.log('[ 2/4 ] capturing standard screenshots...');
  let results;
  try {
    results = await captureAll();
  } finally {
    cleanup();
  }

  console.log('[ 3/4 ] capturing portfolio-filled screenshots...');
  const filledResults = await capturePortfolioFilled();
  results = results.concat(filledResults);

  console.log('[ 4/4 ] done.');
  console.log(`\n${results.length} screenshots saved to ${config.outputDir}/`);
  results.forEach((r) => {
    console.log(`  ${path.basename(r.file)}`);
  });
}

main().catch((err) => {
  console.error('\nError:', err.message);
  process.exit(1);
});
