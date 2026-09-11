import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './ui-tests',
  timeout: 30000,
  workers: 1,
  use: { locale: 'de-DE', baseURL: 'http://127.0.0.1:4178', screenshot: 'only-on-failure' },
  webServer: { command: 'npm run preview -- --host 127.0.0.1 --port 4178', url: 'http://127.0.0.1:4178', reuseExistingServer: false },
});
