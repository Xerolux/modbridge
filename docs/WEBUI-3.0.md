# ModBridge 3.0 WebUI

## Delivered workflows

- Shared loading, error, empty, search and data-health controls across operational pages.
- Refresh failures retain rows and do not advance the last-success timestamp. Polling pauses in hidden tabs, resumes online and aborts pending refreshes when leaving a page.
- Live streams reconnect with backoff, recover after silent connections, and close on navigation. Logs can be searched without losing incoming entries.
- The setup assistant validates and reviews connection settings, creates a disabled proxy, then offers a targeted TCP diagnostic. It never reads or writes Modbus registers. Enable/start remains an explicit action under Control.
- Online updates have their own navigation entry. The UI shows versions and release notes, asks before installation, follows progress across page reloads, and only confirms success after the health endpoint reports the requested version.
- The updater owns its background context, rejects concurrent jobs, verifies SHA256, preserves the previous binary and prepares executable permissions before swapping files.

## Online update requirements

Use a native Linux installation with a writable executable directory and a service manager configured to restart automatically (the provided systemd installer uses `Restart=always`). Docker installations should update their image through their container manager. The application itself does not update container images.

The first upgrade from an older version may require the manual installation procedure if its old updater aborts after the HTTP request ends. The background-context fix is included in 3.0. Backups of replaced binaries remain next to the executable as `.bak.<timestamp>`; this is not a database/config backup or automatic post-restart rollback.

## Compatibility

API and configuration remain compatible. `GET /api/system/diagnostics/connectivity` accepts an optional `proxy_id`; omitted means the existing all-proxy behavior. Unknown IDs return 404. `/api/health` additionally reports the running version.

## Validation and budgets

- `npm run build`, `npm run check:budget`, `npm run test:ui` in `frontend/`.
- 12 browser checks cover all primary routes at 360/768/1440 px, themes, mobile navigation, lazy route recovery, failed refreshes, guided setup, SSE reconnection/cleanup and update confirmation/version verification.
- All production JavaScript combined: maximum 500 KiB gzip; CSS: maximum 40 KiB gzip. Measured locally: 404.4 KiB JS / 23.0 KiB CSS. These totals include lazy routes, not only the initial screen.
- Each primary route must show its heading within 5 seconds in the local preview tests. This is a CI regression threshold, not a latency guarantee for arbitrary hardware or networks.
- Real-backend Playwright coverage includes first-login/password change, proxy management and setup against a loopback TCP listener, proving that diagnostics sends no Modbus payload.
- Go tests cover targeted diagnostics, background updates after request cancellation, concurrent-update rejection, checksum failure and preservation of the previous executable.
- The local Windows GCC/CGO linker cannot produce a functioning standalone backend here; real-backend E2E validation runs in Linux CI. The targeted Go package tests and vet pass locally.

## Release gates

Backend tests, vet/format checks, all Linux builds, frontend budgets, both browser suites and benchmark guardrails must pass before automatic publication. Setting `version.txt` to a new untagged version releases that exact version; subsequent maintenance releases increment the existing build suffix.
