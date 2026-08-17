// Stops the ModBridge instance started by global-setup and removes its data.

import { existsSync, readFileSync, rmSync } from 'node:fs';

export default async function globalTeardown() {
  const stateFile = 'e2e/.server.json';
  if (!existsSync(stateFile)) return;

  const { pid, workDir } = JSON.parse(readFileSync(stateFile, 'utf8'));

  try {
    process.kill(pid, 'SIGTERM');
  } catch {
    // already stopped
  }

  for (const path of [stateFile, 'e2e/.auth.json', workDir]) {
    try {
      rmSync(path, { recursive: true, force: true });
    } catch {
      // best effort
    }
  }
}
