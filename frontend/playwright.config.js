import { defineConfig, devices } from '@playwright/test';

// End-to-end tests run against a real ModBridge binary, started by
// e2e/global-setup.js. Build it first:
//
//   npm run build && rm -rf ../pkg/web/dist && cp -r dist ../pkg/web/dist
//   (cd .. && CGO_ENABLED=1 go build -o modbridge .)
//   npm run test:e2e
//
// Set MODBRIDGE_BIN to point at the binary if it is not ../modbridge, and
// PLAYWRIGHT_CHROMIUM_PATH to reuse a preinstalled Chromium.

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.js',
  globalTeardown: './e2e/global-teardown.js',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1, // One server, one shared config — tests must not race each other
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: process.env.MODBRIDGE_URL || 'http://localhost:8080',
    storageState: 'e2e/.auth.json',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_PATH
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH }
      : {}
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }]
});
