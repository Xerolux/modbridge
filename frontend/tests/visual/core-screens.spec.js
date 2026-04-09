import { test, expect } from '@playwright/test';

const routes = [
  { name: 'login', path: '/#/login' }
];

for (const route of routes) {
  test(`visual ${route.name}`, async ({ page }) => {
    await page.goto(route.path);
    await expect(page).toHaveScreenshot(`${route.name}.png`, {
      fullPage: true,
      maxDiffPixelRatio: 0.03
    });
  });
}
