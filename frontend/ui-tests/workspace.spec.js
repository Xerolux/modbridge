import { test, expect } from '@playwright/test';
const proxies = [{ id: 'test-proxy', name: 'Energy meter · Main building', status: 'Running', listen_addr: ':5020', target_addr: '127.0.0.1:502', requests: 1234, active_connections: 2, tags: [] }];
async function mockApi(page) {
  await page.route('**/api/**', route => {
    const path = new URL(route.request().url()).pathname;
    let data = {};
    if (path === '/api/me') data = { username: 'admin', role: 'admin', permissions: [] };
    if (path === '/api/proxies') data = proxies;
    if (['/api/devices','/api/logs','/api/users','/api/audit/logs'].includes(path)) data = [];
    if (path.endsWith('/stream')) return route.fulfill({ contentType: 'text/event-stream', body: ': connected\n\n' });
    return route.fulfill({ json: data });
  });
}
for (const width of [360, 768, 1440]) {
  test(`pages render and navigation fits at ${width}px`, async ({ page }) => {
    const errors = [];
    page.on('pageerror', error => errors.push(error.stack));
    await page.setViewportSize({ width, height: 900 });
    await mockApi(page);
    for (const path of ['/', '/config', '/control', '/devices', '/logs', '/users', '/audit', '/system', '/setup', '/updates', '/']) {
      const started = Date.now();
      await page.goto(`/#${path}`);
      await expect(page.locator(`.sidebar-link[href="#${path}"]`).first()).toHaveAttribute('aria-current', 'page');
      await expect(page.locator('main')).toBeVisible();
      await expect(page.locator('main h1').first()).toBeVisible();
      expect(Date.now() - started, `route ${path} should render within 5 seconds`).toBeLessThan(5000);
      if (path === '/') {
        await expect(page.locator('.widget-shell')).toHaveCount(1);
        await page.screenshot({ path: `test-results/dashboard-${width}.png`, fullPage: true });
      }
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    }
    if (width < 1024) {
      await page.getByRole('button', { name: 'Navigation öffnen' }).click();
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toHaveCount(0);
      await page.getByRole('button', { name: 'Navigation öffnen' }).click();
      await page.setViewportSize({ width: 1280, height: 900 });
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toHaveCount(0);
      expect(await page.evaluate(() => document.body.style.overflow)).not.toBe('hidden');
    }
    expect(errors).toEqual([]);
  });
}
test('dashboard recovers widgets after initial API failure', async ({ page }) => {
  await mockApi(page);
  let failed = true;
  await page.route('**/api/proxies', route => route.fulfill(failed ? { status: 500, body: 'Unavailable' } : { json: proxies }));
  await page.goto('/#/');
  await expect(page.getByText('Dashboard konnte nicht geladen werden')).toBeVisible();
  failed = false;
  await page.getByRole('button', { name: 'Erneut versuchen' }).click();
  await expect(page.locator('.widget-shell')).toHaveCount(1);
  await expect(page.locator('.widget-shell')).toContainText(proxies[0].name);
});
test('failed lazy page shows recovery without automatic reload loop', async ({ page }) => {
  await mockApi(page);
  await page.goto('/#/');
  await expect(page.locator('.widget-shell')).toHaveCount(1);
  await page.route('**/Config-*.js', route => route.abort());
  await page.locator('.sidebar-link[href="#/config"]').click();
  await expect(page.getByRole('alert')).toContainText('Die Seite konnte nicht geladen werden');
  await expect(page.getByRole('button', { name: 'Seite neu laden' })).toBeVisible();
  await page.unroute('**/Config-*.js');
  await page.getByRole('button', { name: 'Seite neu laden' }).click();
  await expect(page.locator('.config-shell')).toBeVisible();
});

for (const theme of ['light', 'dark', 'bw']) {
  test(`dashboard and settings remain readable in ${theme} theme`, async ({ page }) => {
    await mockApi(page);
    await page.addInitScript(theme => localStorage.setItem('modbridge_theme', theme), theme);
    await page.goto('/#/');
    await expect(page.locator('.widget-shell')).toBeVisible();
    await expect(page.locator('html')).toHaveClass(new RegExp(theme));
    await page.screenshot({ path: `test-results/theme-${theme}.png`, fullPage: true });
    await page.locator('.sidebar-link[href="#/config"]').click();
    await expect(page.locator('.config-shell')).toBeVisible();
    await page.screenshot({ path: `test-results/settings-${theme}.png`, fullPage: true });
  });
}
