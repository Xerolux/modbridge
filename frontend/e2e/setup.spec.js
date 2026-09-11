import { test, expect } from '@playwright/test';
import net from 'node:net';
test('guided setup persists a disabled proxy and diagnoses a real TCP target',async({page})=>{
 let connections=0,bytes=0;
 const target=net.createServer(socket=>{connections++;socket.on('data',data=>{bytes+=data.length});socket.on('error',()=>{});});
 await new Promise(resolve=>target.listen(0,'127.0.0.1',resolve));
 try {
  await page.goto('/#/setup');
  await page.locator('#setup-name').fill('E2E guided setup');
  await page.locator('#setup-listen').fill(':5948');
  await page.locator('#setup-target').fill(`127.0.0.1:${target.address().port}`);
  await page.getByRole('button',{name:/Weiter|Continue/}).click();
  await page.getByRole('button',{name:/Proxy speichern|Save proxy/}).click();
  await expect(page.getByText(/Proxy wurde gespeichert|Proxy saved/)).toBeVisible();expect(connections).toBe(0);
  await page.getByRole('button',{name:/Zielverbindung prüfen|Test target connection/}).click();
  await expect(page.getByText(/TCP-Verbindung erfolgreich|TCP connection succeeded/)).toBeVisible();
  expect(connections).toBe(1);expect(bytes).toBe(0);
  await page.reload();await expect(page.locator('#setup-name')).toBeVisible();
  const response=await page.request.get('/api/proxies');const saved=(await response.json()).find(proxy=>proxy.name==='E2E guided setup');expect(saved.enabled).toBe(false);
 }finally{await new Promise(resolve=>target.close(resolve));}
});
