# Troubleshooting

## Port already in use

```bash
# Find the process occupying the port
sudo lsof -i :8080

# Terminate the process
sudo kill -9 <PID>

# Or use a different port (environment variable)
WEB_PORT=:9090 ./modbridge
```

## Cannot connect to the target device

```bash
# Check reachability
ping 192.168.1.100

# Check the port
nc -zv 192.168.1.100 502
```

## Timeouts despite a reachable device (Home Assistant, SolarEdge & co.)

Typical picture in the client log: individual requests run into a timeout, then every further query fails until the client rebuilds the connection. In `pymodbus`-based integrations it looks like this:

```
Error reading inverter ID 4 at InverterCommon:
Response timeout after 3 seconds for transaction with ID 0x23
```

Three causes that ModBridge counters specifically:

1. **Multiple sessions to the device.** Many inverters (SolarEdge/SunSpec, small RTU gateways) serve only one Modbus connection and silently ignore further ones. `max_target_conns: 1` enforces exactly one connection to the target device; requests from multiple clients are queued in front of it.
2. **Requests too close together.** Some devices discard requests that follow each other without a pause. `min_request_gap_ms` (e.g. `100`) enforces a minimum gap.
3. **The answer arrives after the client has given up.** If forwarding including retries takes longer than the client's timeout, the late response lands on its next request — from then on no transaction ID fits anymore and every query fails. `request_timeout_ms` caps the entire request; when the budget is used up, ModBridge answers with a regular Modbus exception (`0x0B`, *Gateway Target Device Failed To Respond*) instead of late with payload.

Additionally, ModBridge assigns its own transaction IDs towards the target device and discards responses that do not belong to the current request. How often that happens is shown as `stale_responses` in the proxy status — persistently rising values mean the device answers more slowly than the timeouts allow.

### When the device is fundamentally slower than the client waits

With a SolarEdge leader with followers (unit IDs 2, 3, 4 …), the follower registers travel across the RS485 chain and often need more than the 3 s Home Assistant waits. Typical pattern: the leader answers reliably, the followers run into timeouts.

A shorter budget does not help here — the opposite does: cache plus background polling:

```json
"max_target_conns": 1,
"min_request_gap_ms": 250,
"read_timeout": 15,
"max_retries": 1,
"request_timeout_ms": 0,
"cache_enabled": true,
"cache_ttl_ms": 20000,
"poll_interval_ms": 5000
```

ModBridge then queries the registers by itself every 5 s and serves Home Assistant instantly from the cache. The value is thereby up to 5 s old — uncritical for PV data. The **SolarEdge Leader + Follower** profile sets exactly this.

In the proxy dialog of the web interface, the device profile **SolarEdge / SunSpec** sets these values with one click; for Huawei inverters and sDongles there is a dedicated profile. The profiles only fill the form — existing proxies stay unchanged until a profile is selected there.

Recommended starting point for a SolarEdge inverter with multiple unit IDs, queried from Home Assistant (client timeout there: 3 s):

```json
{
  "max_target_conns": 1,
  "min_request_gap_ms": 100,
  "request_timeout_ms": 2500,
  "max_retries": 1,
  "read_timeout": 2
}
```

## Queries take tens of seconds although the network is healthy

Typical picture: a connectivity test answers in under a second, but a full query takes 30 s. That is not a contradiction — the test makes a handful of round trips, the query several hundred.

Do the math instead of guessing. The proxy status shows `latency_p50_ms` and `latency_p95_ms`; multiplied by the number of requests per query cycle this gives the expected duration. If that roughly matches the observed time, it is plain arithmetic and not a hung device.

The largest single factor is often `min_request_gap_ms`, because the cost applies **per request**:

| Gap | 50 requests | 100 requests | 300 requests |
|-----|------------|--------------|--------------|
| 50 ms | 2.5 s | 5 s | 15 s |
| 100 ms | 5 s | 10 s | 30 s |
| 250 ms | 12.5 s | 25 s | 75 s |

Proceed in this order:

1. **Check the gap and lower it step by step** (250 → 100 → 50 ms). Watch `stale_responses` and the error count: if they rise, the gap was needed and the last working value applies.
2. **Enable the cache and the background poller.** This makes the query duration at the client nearly independent of the device, because it is served from the cache.
3. **Increase the client's query interval.** As long as a query takes longer than the interval, one is permanently running — the device never comes to rest and every single request gets slower. That is a feedback loop that reinforces itself.
4. **Check who else accesses the proxy.** `active_connections` in the proxy status shows it. A second client with a similar frequency adds its load to the same device.

While the background poller is running, ModBridge also logs when a refresh round takes longer than the configured interval. That message means the same as point 3, just on the proxy side: queries are running continuously and the values in the cache are older than the interval suggests.

## Forgot the admin password

### Reset username and password via the WebUI

Stop ModBridge and enable the WebUI recovery once:

```bash
sudo systemctl stop modbridge.service
sudo -u modbridge ./modbridge --enable-account-recovery
sudo systemctl start modbridge.service
```

Then open the login page and choose **"Forgot credentials?"**. With the printed recovery code (valid for 15 minutes) you can set a new username and password. The code can be used only once.

If multiple administrators exist, specify the target account explicitly:

```bash
sudo -u modbridge ./modbridge --enable-account-recovery --recovery-user existing-name
```

### Reset only the password via the console

Stop ModBridge and generate a new one-time password for the admin user locally:

```bash
sudo systemctl stop modbridge.service
sudo -u modbridge ./modbridge --reset-password admin
sudo systemctl start modbridge.service
```

The command prints a random one-time password. It must be changed immediately after login. Run this command only locally on the ModBridge host.

## Docker container does not start

```bash
docker logs modbridge
docker ps -a
```

## systemd service problems

```bash
sudo bash scripts/modbridge.sh status
journalctl -u modbridge.service -n 100
```
