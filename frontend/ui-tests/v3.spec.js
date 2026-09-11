import { test, expect } from '@playwright/test';
async function api(page) {
 await page.route('**/api/**', route => {
  const path=new URL(route.request().url()).pathname;
  const data=path==='/api/me'?{username:'admin',role:'admin',permissions:[]}:path==='/api/proxies'?[]:{};
  return route.fulfill({json:data});
 });
}
test('setup saves a disabled proxy and probes only that proxy',async({page})=>{
 await page.setViewportSize({width:390,height:844});
 await api(page);let created;let probes=0;
 await page.route('**/api/proxies',route=>{if(route.request().method()==='POST'){created=route.request().postDataJSON();return route.fulfill({status:201,json:{id:'wizard'}})}return route.fulfill({json:[]})});
 await page.route('**/api/system/diagnostics/connectivity?*',route=>{probes++;expect(new URL(route.request().url()).searchParams.get('proxy_id')).toBe('wizard');return route.fulfill({json:{wizard:{reachable:true}}})});
 await page.goto('/#/setup');
 await expect(page.getByRole('button',{name:'Weiter'})).toBeDisabled();
 await page.locator('#setup-name').fill('Test meter');await page.locator('#setup-target').fill('127.0.0.1:502');
 await page.getByRole('button',{name:'Weiter'}).click();expect(created).toBeUndefined();
 await page.getByRole('button',{name:'Proxy speichern'}).click();await expect(page.getByText('Proxy wurde gespeichert')).toBeVisible();
 expect(created.enabled).toBe(false);expect(probes).toBe(0);
 await page.getByRole('button',{name:'Zielverbindung prüfen'}).click();await expect(page.getByText(/TCP-Verbindung erfolgreich/)).toBeVisible();expect(probes).toBe(1);await page.screenshot({path:'test-results/setup-mobile.png',fullPage:true});
});
test('failed refresh preserves rows and last successful timestamp',async({page})=>{
 await api(page);let fail=false;
 await page.route('**/api/devices',route=>route.fulfill(fail?{status:500,body:'offline'}:{json:[{ip:'127.0.0.1',name:'Meter',request_count:3}]}));
 await page.goto('/#/devices');await expect(page.locator('input').filter({visible:true}).first()).toBeVisible();
 await page.locator('.data-health button').click();await expect(page.locator('.data-health time')).toBeVisible();const time=await page.locator('.data-health time').getAttribute('datetime');
 fail=true;await page.locator('.data-health button').click();await expect(page.locator('.data-health')).toContainText('Aktualisierung fehlgeschlagen');
 await expect(page.locator('.data-health time')).toHaveAttribute('datetime',time);expect(await page.locator('input').evaluateAll(inputs => inputs.some(input => input.value === 'Meter'))).toBe(true);
 fail=false;await page.locator('.data-health button').click();await expect(page.locator('.data-health')).toContainText('Daten verfügbar');
});
test('online update requires confirmation and verifies the running version',async({page})=>{
 await page.setViewportSize({width:390,height:844});
 await api(page);let posts=0;let installed=false;
 await page.route('**/api/update/check',r=>r.fulfill({json:{current_version:installed?'3.0.0':'2.0.10.22',latest_version:'3.0.0',update_available:!installed,os:'linux',arch:'amd64'}}));
 await page.route('**/api/update/perform',r=>{posts++;return r.fulfill({json:{job_started:true}})});
 await page.route('**/api/update/status',r=>r.fulfill({json:{state:'done',progress:100}}));
 await page.route('**/api/health',r=>r.fulfill({json:{status:'ok',version:installed?'3.0.0':'2.0.10.22'}}));
 await page.goto('/#/updates');await page.screenshot({path:'test-results/updates-mobile.png',fullPage:true});await page.getByRole('button',{name:/Update installieren|Install update/i}).first().click();expect(posts).toBe(0);
 await page.getByRole('dialog').getByRole('button',{name:/Update installieren|Install update/i}).click();await expect.poll(()=>posts).toBe(1);
 await expect(page.getByText('Update abgeschlossen: Die neue Version läuft.')).toHaveCount(0);
 installed=true;await expect(page.getByText('Update abgeschlossen: Die neue Version läuft.')).toBeVisible({timeout:12000});expect(posts).toBe(1);
});

test('dashboard reconnects live updates and closes streams on navigation',async({page})=>{
 await api(page);
 await page.route('**/api/proxies',r=>r.fulfill({json:[{id:'live',name:'Live meter',status:'Running',requests:1}]}));
 await page.addInitScript(()=>{
  window.testStreams=[];
  window.EventSource=class extends EventTarget {
   static CLOSED=2;
   constructor(){super();this.closed=false;window.testStreams.push(this);setTimeout(()=>{if(!this.closed)this.onopen?.({})},0)}
   close(){this.closed=true}
  };
 });
 await page.goto('/#/');await expect(page.locator('.widget-shell')).toContainText('Live meter');
 await page.evaluate(()=>window.testStreams.at(-1).onmessage({data:JSON.stringify({type:'proxy_updated',proxy:{id:'live',name:'Live meter',status:'Running',requests:42}})}));
 await expect(page.locator('.widget-shell')).toContainText('42');
 await page.evaluate(()=>window.testStreams.at(-1).onerror({target:{readyState:0}}));
 await expect.poll(()=>page.evaluate(()=>window.testStreams.length)).toBeGreaterThan(1);
 await page.evaluate(()=>window.testStreams.at(-1).onmessage({data:JSON.stringify({type:'proxy_updated',proxy:{id:'live',name:'Live meter',status:'Running',requests:84}})}));
 await expect(page.locator('.widget-shell')).toContainText('84');
 await page.locator('.sidebar-link[href="#/config"]').click();await expect(page.locator('.config-shell')).toBeVisible();
 expect(await page.evaluate(()=>window.testStreams.every(stream=>stream.closed))).toBe(true);
});
