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
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toBeVisible();
      await expect(page.locator('body')).toHaveCSS('overflow', 'hidden');
      await page.setViewportSize({ width: 1280, height: 900 });
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toHaveCount(0);
      await expect(page.locator('body')).not.toHaveCSS('overflow', 'hidden');
      await expect(page.locator('main')).toBeFocused();
      await page.setViewportSize({ width, height: 900 });
      await expect(page.getByRole('dialog', { name: 'Hauptnavigation' })).toHaveCount(0);
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

test('dashboard shares the initial proxy request and preserves the last successful result on refresh failure', async ({ page }) => {
  await mockApi(page);
  let requests = 0;
  let failed = false;
  await page.route('**/api/proxies', async route => {
    requests++;
    await new Promise(resolve => setTimeout(resolve, 150));
    return route.fulfill(failed ? { status: 500 } : { json: proxies });
  });
  await page.goto('/#/');
  await expect(page.locator('.widget-shell')).toHaveCount(1);
  const status = page.locator('main .data-health');
  const lastSuccess = status.locator('time');
  await expect(lastSuccess).toBeVisible();
  expect(requests).toBe(1);
  const timestamp = await lastSuccess.getAttribute('datetime');

  failed = true;
  await status.getByRole('button').click();
  await expect(status.getByRole('button')).toBeDisabled();
  await expect(status).toContainText('Aktualisierung fehlgeschlagen');
  await expect(status.getByRole('button')).toBeEnabled();
  await expect(lastSuccess).toHaveAttribute('datetime', timestamp);
  await expect(page.locator('.widget-shell')).toContainText(proxies[0].name);
});

test('security settings remove only the wildcard and show when a restart is required', async ({ page }) => {
  await mockApi(page);
  let saved;
  await page.route('**/api/config/system', route => {
    if (route.request().method() === 'PUT') {
      saved = route.request().postDataJSON();
      return route.fulfill({ json: {} });
    }
    return route.fulfill({ json: { cors_allowed_origins: ['*', 'https://panel.example.net:8443'] } });
  });
  await page.setViewportSize({ width: 360, height: 900 });
  await page.goto('/#/config');
  await page.getByRole('tab', { name: 'Sicherheit' }).click();
  await expect(page.getByLabel('Erlaubte Origins', { exact: true })).toBeVisible();
  await expect(page.getByText('Mit * sind Browser-Anfragen', { exact: false })).toBeVisible();
  await page.getByRole('button', { name: 'Platzhalter entfernen' }).click();
  await expect(page.getByText('Mit * sind Browser-Anfragen', { exact: false })).toHaveCount(0);
  expect(saved).toBeUndefined();
  await page.getByRole('button', { name: 'Sicherheitskonfiguration speichern' }).click();
  await expect(page.getByText('Starte ModBridge neu', { exact: false })).toBeVisible();
  expect(saved.cors_allowed_origins).toEqual(['https://panel.example.net:8443']);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
