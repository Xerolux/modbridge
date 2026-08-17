// Starts a real ModBridge instance for the end-to-end tests.
//
// The binary is used as a user gets it: first start prints a generated admin
// password and demands a password change. This setup walks that flow once,
// stores the resulting session, and leaves the server running for the specs.

import { spawn } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { chromium } from '@playwright/test';

const BINARY = process.env.MODBRIDGE_BIN
  ? resolve(process.env.MODBRIDGE_BIN)
  : resolve(process.cwd(), '..', 'modbridge');
const BASE_URL = process.env.MODBRIDGE_URL || 'http://localhost:8080';
const NEW_PASSWORD = 'Zt7#qLm2$vRx9Kd!';

const launchOptions = process.env.PLAYWRIGHT_CHROMIUM_PATH
  ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH }
  : {};

async function waitForServer(url, timeoutMs = 45000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url, { redirect: 'manual' });
      if (res.status > 0) return;
    } catch {
      // not up yet
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`server at ${url} did not come up within ${timeoutMs}ms`);
}

export default async function globalSetup(config) {
  const workDir = mkdtempSync(join(tmpdir(), 'modbridge-e2e-'));
  const server = spawn(BINARY, [], { cwd: workDir, stdio: ['ignore', 'pipe', 'pipe'] });

  let output = '';
  let initialPassword = null;
  const capture = (chunk) => {
    output += chunk.toString();
    const match = output.match(/Initial admin password:\s*(\S+)/);
    if (match) initialPassword = match[1];
  };
  server.stdout.on('data', capture);
  server.stderr.on('data', capture);

  const stop = () => {
    try {
      server.kill('SIGTERM');
    } catch {
      // already gone
    }
    try {
      rmSync(workDir, { recursive: true, force: true });
    } catch {
      // best effort
    }
  };

  try {
    await waitForServer(BASE_URL);

    const deadline = Date.now() + 15000;
    while (!initialPassword && Date.now() < deadline) {
      await new Promise((r) => setTimeout(r, 200));
    }
    if (!initialPassword) {
      throw new Error(`server never printed an initial admin password.\nOutput:\n${output}`);
    }

    const browser = await chromium.launch(launchOptions);
    const page = await browser.newPage();

    // Wait for the app to leave a route rather than for a fixed number of
    // seconds. A loaded CI runner is slower than a developer's machine, and a
    // sleep that is long enough locally is a coin flip there.
    const leaveRoute = (route) =>
      page.waitForFunction(
        (r) => !window.location.hash.includes(r),
        route,
        { timeout: 30000 }
      );

    const login = async (password) => {
      await page.fill('#login-username', 'admin');
      await page.fill('#login-password', password);
      await page.getByRole('button', { name: /anmelden|login|sign in/i }).click();
      await leaveRoute('/login');
    };

    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await login(initialPassword);

    if (page.url().includes('change-password')) {
      await page.fill('#cp-current', initialPassword);
      await page.fill('#cp-new', NEW_PASSWORD);
      await page.fill('#cp-confirm', NEW_PASSWORD);
      await page.getByRole('button', { name: /change password|passwort ändern/i }).click();
      await leaveRoute('/change-password');

      // Some builds send you back to the login form afterwards.
      if (page.url().includes('/login')) {
        await login(NEW_PASSWORD);
      }
    }

    if (page.url().includes('login') || page.url().includes('change-password')) {
      // Say why, so a CI failure here is diagnosable without a rerun.
      const messages = await page
        .locator('.p-message, .login-error, [role=alert], .p-toast-detail')
        .allTextContents()
        .catch(() => []);
      await page.screenshot({ path: 'e2e-setup-failure.png', fullPage: true }).catch(() => {});
      throw new Error(
        `login did not complete, ended up at ${page.url()}` +
          (messages.length ? `\nPage messages: ${messages.join(' | ')}` : '')
      );
    }

    await page.context().storageState({ path: 'e2e/.auth.json' });
    await browser.close();
  } catch (err) {
    stop();
    throw err;
  }

  // Hand the teardown the pieces it needs to shut the instance down again.
  writeFileSync('e2e/.server.json', JSON.stringify({ pid: server.pid, workDir }));
  server.unref();
}
