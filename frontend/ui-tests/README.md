# WebUI regression checks

Run `npm run build` followed by `npm run test:ui` from `frontend/`.
The tests start a local production preview on port 4178 and intercept API requests.
They do not require a backend, credentials, or Modbus hardware.

Coverage: primary routes at 360/768/1440 px, mobile drawer keyboard and resize
behavior, dashboard recovery after an API error, failed lazy route recovery,
and light/dark/monochrome rendering. Screenshots are written to `test-results/`.
The existing `npm run test:e2e` suite separately exercises a real local backend.
