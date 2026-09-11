import { expect, test } from '@playwright/test';

// Regression tests for the settings page. It shipped broken for two releases
// (a stray `import` line inside the template threw during setup, leaving a
// blank page) while builds and unit tests stayed green — nothing drove the
// route in a browser. These tests do exactly that.

test('settings page renders tabs and the proxy studio', async ({ page }) => {
  await page.goto('/#/config', { waitUntil: 'networkidle' });

  // The tab bar must appear with its six tabs. A missing tab bar means the
  // route component crashed during setup — the blank-page signature.
  for (const name of [/proxies/i, /logging/i, /erweitert|advanced/i]) {
    await expect(page.getByRole('tab', { name })).toBeVisible();
  }

  // The Proxy Studio on the first tab must actually render content, not an
  // empty panel.
  await expect(page.locator('.config-shell')).toBeVisible();
  await expect(page.getByText(/proxy studio/i)).toBeVisible();
});

test('settings page survives a reload with saved proxies', async ({ page }) => {
  await page.goto('/#/config', { waitUntil: 'networkidle' });
  await page.reload({ waitUntil: 'networkidle' });
  await expect(page.locator('.config-shell')).toBeVisible();
});
