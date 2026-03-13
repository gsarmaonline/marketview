const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');

const BASE_URL = process.env.NEXT_PUBLIC_URL || 'http://localhost:3002';
const SCREENSHOTS_DIR = path.join(__dirname, '..', 'screenshots');

const routes = [
  { path: '/', name: 'home' },
  { path: '/stock/RELIANCE', name: 'stock-reliance' },
];

const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'mobile', width: 390, height: 844 },
];

async function waitForServer(url, retries = 20, delayMs = 1500) {
  const http = require('http');
  for (let i = 0; i < retries; i++) {
    try {
      await new Promise((resolve, reject) => {
        http.get(url, (res) => resolve(res)).on('error', reject);
      });
      return;
    } catch {
      if (i < retries - 1) {
        process.stdout.write(`\rWaiting for server... (${i + 1}/${retries})`);
        await new Promise((r) => setTimeout(r, delayMs));
      }
    }
  }
  throw new Error(`Server at ${url} did not become ready in time`);
}

async function takeScreenshots() {
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });

  console.log('Waiting for dev server...');
  await waitForServer(BASE_URL);
  console.log('\nServer ready.');

  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const page = await browser.newPage();
  let captured = 0;
  let failed = 0;

  for (const route of routes) {
    for (const viewport of viewports) {
      await page.setViewport({ width: viewport.width, height: viewport.height });

      try {
        await page.goto(`${BASE_URL}${route.path}`, {
          waitUntil: 'networkidle2',
          timeout: 30000,
        });

        // Wait for loading skeletons to disappear
        await page.waitForFunction(
          () => !document.querySelector('[style*="animation: pulse"]'),
          { timeout: 10000 }
        ).catch(() => {});

        await new Promise((r) => setTimeout(r, 500));

        const filename = `${route.name}-${viewport.name}.png`;
        const filepath = path.join(SCREENSHOTS_DIR, filename);
        await page.screenshot({ path: filepath, fullPage: true });
        console.log(`  captured ${filename}`);
        captured++;
      } catch (err) {
        console.error(`  FAILED ${route.name}-${viewport.name}: ${err.message}`);
        failed++;
      }
    }
  }

  await browser.close();

  console.log(`\nDone: ${captured} captured, ${failed} failed`);
  console.log(`Saved to: ${SCREENSHOTS_DIR}`);
}

takeScreenshots().catch((err) => {
  console.error('Fatal:', err.message);
  process.exit(1);
});
