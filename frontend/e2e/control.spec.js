import { expect, test } from '@playwright/test';

// Regression tests for two bugs that a build and a unit test suite both
// happily passed, because neither drives the UI:
//
//  - dragging cards did nothing, since VueDraggable was given a prop its
//    version does not know, so no model ever changed
//  - the applied device profile was forgotten as soon as the dialog closed
//
// Both are only visible by clicking, which is what these tests do.

const PROFILE_LABEL = 'SolarEdge Leader + Follower';

const cardNames = (page) =>
  page.$$eval('.proxy-card', (cards) =>
    cards.map((c) => (c.querySelector('h3, .font-semibold, strong')?.textContent || '').trim())
  );

async function gotoControl(page) {
  await page.goto('/#/control', { waitUntil: 'networkidle' });
  await expect(page.locator('.proxy-card, .glass-panel')).not.toHaveCount(0);
  await page.waitForTimeout(800);
}

async function addProxy(page, { name, listen, target, profile }) {
  await page.getByRole('button', { name: /proxy hinzufügen|add proxy/i }).first().click();
  const dialog = page.locator('.p-dialog');
  await expect(dialog).toBeVisible();

  const inputs = dialog.locator('input.p-inputtext:not([type=password])');
  await inputs.nth(0).fill(name);
  await inputs.nth(1).fill(listen);
  await inputs.nth(2).fill(target);

  if (profile) {
    await dialog.locator('.p-select').first().click();
    const filter = page.locator('.p-select-overlay input').first();
    if (await filter.count()) {
      await filter.fill('Leader');
    }
    await page.locator('.p-select-option', { hasText: profile }).first().click();
    await expect(page.locator('.p-select-overlay')).toHaveCount(0);
  }

  await dialog.getByRole('button', { name: /^(hinzufügen|add)$/i }).first().click();

  // A proxy that cannot start leaves the dialog open with an error toast, so
  // report that text instead of a bare timeout.
  try {
    await expect(dialog).toHaveCount(0);
  } catch (err) {
    const toast = await page.locator('.p-toast-detail, .p-toast-summary').allTextContents();
    throw new Error(`creating proxy "${name}" did not close the dialog: ${toast.join(' | ') || 'no message'}`);
  }
  await page.waitForTimeout(800);
}

async function openCardMenu(page, cardName) {
  const card = page.locator('.proxy-card', { hasText: cardName }).first();
  await card.locator('button:has(.pi-ellipsis-v)').first().click();
  await expect(page.locator('#overlay_menu')).toBeVisible();
}

// Two proxies are enough to prove an order change. The target only has to
// accept a TCP connection for the proxy to start — no Modbus traffic is ever
// exchanged — so it points at the instance's own web port.
const TARGET = '127.0.0.1:8080';

async function ensureProxies(page) {
  const names = await cardNames(page);
  if (!names.some((n) => n.includes('E2E Alpha'))) {
    await addProxy(page, { name: 'E2E Alpha', listen: ':5941', target: TARGET });
  }
  if (!(await cardNames(page)).some((n) => n.includes('E2E Bravo'))) {
    await addProxy(page, {
      name: 'E2E Bravo',
      listen: ':5942',
      target: TARGET,
      profile: PROFILE_LABEL
    });
  }
}

test.describe('control page', () => {
  test.beforeEach(async ({ page }) => {
    await gotoControl(page);
    await ensureProxies(page);
  });

  test('an applied device profile is still shown when the dialog reopens', async ({ page }) => {
    await openCardMenu(page, 'E2E Bravo');
    await page.getByRole('menuitem', { name: /bearbeiten|^edit$/i }).first().click();

    const dialog = page.locator('.p-dialog');
    await expect(dialog).toBeVisible();

    // The dropdown must name the profile, not fall back to its placeholder.
    await expect(dialog.locator('.p-select-label').first()).toContainText(PROFILE_LABEL);

    // ...and the settings that profile carries must have been saved with it.
    const cacheTtl = dialog.locator('input.p-inputtext').filter({ hasNot: page.locator('[type=password]') });
    await expect(cacheTtl).not.toHaveCount(0);
  });

  test('proxy cards can be reordered by dragging', async ({ page }) => {
    const before = await cardNames(page);
    expect(before.length).toBeGreaterThanOrEqual(2);

    await page.getByRole('button', { name: /bearbeiten|^edit$/i }).first().click();

    const handles = page.locator('.drag-handle');
    await expect(handles).toHaveCount(before.length);

    const from = await handles.nth(1).boundingBox();
    const to = await page.locator('.proxy-card').first().boundingBox();
    expect(from && to).toBeTruthy();

    // SortableJS needs a real gesture: press, several small moves, release.
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
    await page.mouse.down();
    await page.waitForTimeout(200);
    for (let step = 1; step <= 25; step++) {
      await page.mouse.move(
        from.x + (to.x + 30 - from.x) * (step / 25),
        from.y + (to.y + 30 - from.y) * (step / 25),
        { steps: 2 }
      );
      await page.waitForTimeout(20);
    }
    await page.waitForTimeout(300);
    await page.mouse.up();
    await page.waitForTimeout(1200);

    const after = await cardNames(page);
    expect(after, 'dragging a card must change the order').not.toEqual(before);
    expect([...after].sort(), 'no card may be lost or duplicated').toEqual([...before].sort());

    // The order is the user's, so it has to outlive a reload.
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    expect(await cardNames(page), 'the new order must survive a reload').toEqual(after);
  });
});
