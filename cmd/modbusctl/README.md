# modbusctl - Modbridge CLI Tool

`modbusctl` is a command-line interface for managing Modbridge, the enterprise-grade Modbus TCP proxy.

## Installation

### Build from Source

```bash
cd cmd/modbusctl
go build -o modbusctl
sudo mv modbusctl /usr/local/bin/
```

### Verify Installation

```bash
modbusctl --version
```

## Configuration

### API URL

By default, modbusctl connects to `http://localhost:8080`. You can override this:

```bash
# Using flag
modbusctl --api-url http://modbridge.example.com:8080 proxy list

# Using environment variable
export MODBRIDGE_API_URL=http://modbridge.example.com:8080
modbusctl proxy list

# Using config file (~/.modbusctl.yaml)
cat > ~/.modbusctl.yaml <<EOF
api-url: http://modbridge.example.com:8080
output: table
verbose: false
EOF
```

### Output Formats

```bash
# Table format (default, human-readable)
modbusctl proxy list

# JSON format (for scripting)
modbusctl proxy list --output json

# YAML format
modbusctl proxy list --output yaml
```

## Commands

### Proxy Management

#### List All Proxies

```bash
modbusctl proxy list
```

Output:
```
ID              NAME            LISTEN          TARGET          STATUS    REQUESTS  ERRORS
────────────────────────────────────────────────────────────────────────────────────────────
plc-gateway     PLC Gateway     :5020          192.168.1.10:502   running   1234      5
scada-proxy     SCADA Proxy     :5021          10.0.0.5:502       running   5678      12
```

#### Get Proxy Details

```bash
modbusctl proxy get plc-gateway
```

Output:
```
Proxy: PLC Gateway
  ID:           plc-gateway
  Listen Addr:  :5020
  Target Addr:  192.168.1.10:502
  Status:       running
  Enabled:      true

Statistics:
  Requests:     1234
  Errors:       5
  Uptime:       86400 seconds
```

#### Create a New Proxy

```bash
modbusctl proxy create \
  --name "Production PLC" \
  --listen ":5022" \
  --target "192.168.1.20:502" \
  --pool-size 20 \
  --pool-min 5
```

#### Start/Stop/Restart Proxy

```bash
modbusctl proxy start plc-gateway
modbusctl proxy stop plc-gateway
modbusctl proxy restart plc-gateway
```

#### Delete a Proxy

```bash
modbusctl proxy delete plc-gateway
```

### Device Management

#### List Discovered Devices

```bash
modbusctl device list
```

Output:
```
DEVICE ID    NAME         ADDRESS    PROXY         STATUS    LAST SEEN            REQUESTS
──────────────────────────────────────────────────────────────────────────────────────────────
device-1     PLC Unit 1   1          plc-gateway   online    2024-01-15 10:30:45  234
device-2     PLC Unit 2   2          plc-gateway   online    2024-01-15 10:30:50  567
```

#### Get Device Details

```bash
modbusctl device get device-1
```

#### Rename a Device

```bash
modbusctl device rename device-1 --name "Production PLC #1"
```

#### View Device History

```bash
# Last 24 hours (default)
modbusctl device history device-1

# Last 7 days
modbusctl device history device-1 --since 7d

# Limit to 50 entries
modbusctl device history device-1 --limit 50
```

### Metrics & Monitoring

#### System Overview

```bash
modbusctl metrics overview
```

Output:
```
=== System Overview ===
Uptime:              24h 15m 30s
Total Requests:      123456
Total Errors:        234
Error Rate:          0.19%
Active Connections:  45
Active Proxies:      3

=== Performance ===
Avg Latency:         2.34 ms
P50 Latency:         1.89 ms
P95 Latency:         5.67 ms
P99 Latency:         12.34 ms
Requests/sec:        142.56

=== Resource Usage ===
CPU Usage:           15.3%
Memory Usage:        512 MB / 2 GB (25.0%)
Goroutines:          234
```

#### Proxy Metrics

```bash
# All proxies
modbusctl metrics proxy

# Specific proxy
modbusctl metrics proxy --proxy plc-gateway
```

#### System Resources

```bash
modbusctl metrics system
```

#### Health Check

```bash
modbusctl metrics health
```

### Backup & Restore

#### List Backups

```bash
modbusctl backup list
```

#### Create a Backup

```bash
# Auto-generated name
modbusctl backup create

# Custom name
modbusctl backup create --name "pre-upgrade-backup"
```

#### Restore from Backup

```bash
modbusctl backup restore backup-20240115-103045 --confirm
```

⚠️ **Warning:** Restore operations overwrite current configuration!

#### Export Backup to File

```bash
modbusctl backup export backup-20240115-103045 -o backup.tar.gz
```

#### Import Backup from File

```bash
modbusctl backup import -f backup.tar.gz
```

#### Delete a Backup

```bash
modbusctl backup delete backup-20240115-103045
```

### User Management (Admin Only)

#### List Users

```bash
modbusctl user list
```

#### Get User Details

```bash
modbusctl user get john.doe
```

#### Create a User

```bash
modbusctl user create \
  --username john.doe \
  --password secretpass123 \
  --email john.doe@example.com \
  --role operator
```

Roles:
- `admin` - Full access to all operations
- `operator` - Can manage proxies and devices
- `viewer` - Read-only access

#### Update User

```bash
# Change role
modbusctl user update john.doe --role admin

# Change email
modbusctl user update john.doe --email newemail@example.com

# Disable user
modbusctl user update john.doe --enabled=false
```

#### Change Password

```bash
modbusctl user password john.doe --password newpassword123
```

#### Delete User

```bash
modbusctl user delete john.doe
```

### Audit Logs

#### List Recent Logs

```bash
# Last 24 hours (default)
modbusctl audit list

# Last 7 days
modbusctl audit list --since 7d

# Filter by user
modbusctl audit list --user admin --since 24h

# Filter by action
modbusctl audit list --action proxy.create --since 1h
```

#### Search Logs

```bash
modbusctl audit search --query "delete" --since 7d
```

#### Export Logs

```bash
# Export as JSON
modbusctl audit export -o audit-logs.json --since 30d

# Export as CSV
modbusctl audit export -o audit-logs.csv --since 30d --format csv
```

### Configuration Validation

#### Validate Config File

```bash
# Basic validation
modbusctl validate config -f config.json

# Strict validation (additional checks)
modbusctl validate config -f config.json --strict
```

Example output:
```
Validating configuration file: config.json

✅ JSON syntax is valid

Validating 2 proxies...

Proxy 'plc-gateway':
  ✅ Listen address: :5020
  ✅ Target address: 192.168.1.10:502
  ✅ Connection pool: size=10, min=2
  ✅ Enabled: true

Proxy 'scada-proxy':
  ✅ Listen address: :5021
  ✅ Target address: 10.0.0.5:502
  ✅ Connection pool: size=15, min=3
  ✅ Enabled: true

Validating server settings...
  ✅ API server address: :8080
  ✅ TLS enabled
  ✅ Certificate file: /path/to/cert.pem
  ✅ Key file: /path/to/key.pem

==================================================
Validation Summary:
  Errors:   0
  Warnings: 0

✅ Configuration is valid
```

#### Test Proxy Connection

```bash
modbusctl validate proxy \
  --listen ":5022" \
  --target "192.168.1.20:502" \
  --timeout 5s
```

Example output:
```
Validating proxy configuration...

Testing listen address: :5022
  ✅ Can bind to :5022

Testing target address: 192.168.1.20:502
  Attempting connection (timeout: 5s)...
  ✅ Successfully connected to 192.168.1.20:502

✅ Proxy configuration is valid
```

## Scripting Examples

### Bash Script: Bulk Proxy Management

```bash
#!/bin/bash
# Create multiple proxies from a list

while IFS=',' read -r name listen target; do
  echo "Creating proxy: $name"
  modbusctl proxy create \
    --name "$name" \
    --listen "$listen" \
    --target "$target" \
    --pool-size 10
done < proxies.csv
```

### JSON Processing with jq

```bash
# Get all running proxies
modbusctl proxy list --output json | jq '.[] | select(.status == "running") | .id'

# Get total request count across all proxies
modbusctl proxy list --output json | jq '[.[].stats.requests] | add'

# Find proxies with error rate > 5%
modbusctl proxy list --output json | jq '.[] | select((.stats.errors / .stats.requests) > 0.05)'
```

### Python Script: Monitor Metrics

```python
#!/usr/bin/env python3
import subprocess
import json
import time

while True:
    # Get metrics as JSON
    result = subprocess.run(
        ['modbusctl', 'metrics', 'overview', '--output', 'json'],
        capture_output=True,
        text=True
    )

    metrics = json.loads(result.stdout)

    # Check error rate
    if metrics['error_rate'] > 0.05:
        print(f"⚠️  High error rate: {metrics['error_rate']*100:.2f}%")

    # Check CPU usage
    if metrics['resources']['cpu_percent'] > 80:
        print(f"⚠️  High CPU usage: {metrics['resources']['cpu_percent']:.1f}%")

    time.sleep(60)  # Check every minute
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MODBRIDGE_API_URL` | API server URL | `http://localhost:8080` |
| `MODBRIDGE_CONFIG` | Config file path | `$HOME/.modbusctl.yaml` |
| `MODBRIDGE_OUTPUT` | Output format | `table` |
| `MODBRIDGE_VERBOSE` | Enable verbose output | `false` |

## Troubleshooting

### Connection Issues

```bash
# Verify API server is running
curl http://localhost:8080/health

# Test with verbose output
modbusctl --verbose proxy list

# Use different API URL
modbusctl --api-url http://192.168.1.100:8080 proxy list
```

### Authentication

If your Modbridge instance requires authentication, you can use environment variables or config file:

```yaml
# ~/.modbusctl.yaml
api-url: http://localhost:8080
auth:
  username: admin
  password: secretpass
```

Or use HTTP Basic Auth in the URL:

```bash
modbusctl --api-url http://admin:secretpass@localhost:8080 proxy list
```

## Advanced Usage

### Watch Mode (using watch command)

```bash
# Monitor proxy list (refresh every 2 seconds)
watch -n 2 modbusctl proxy list

# Monitor system metrics
watch -n 5 modbusctl metrics overview
```

### Combining Commands

```bash
# Create backup before making changes
modbusctl backup create --name "pre-change" && \
  modbusctl proxy create --name "new-proxy" --listen ":5023" --target "192.168.1.30:502"

# Restart all proxies
for proxy in $(modbusctl proxy list --output json | jq -r '.[].id'); do
  modbusctl proxy restart "$proxy"
done
```

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Invalid arguments
- `3` - API connection error
- `4` - Authentication error
- `5` - Validation error

## Support

For issues and feature requests, visit:
https://github.com/yourusername/modbridge/issues

## License

See the main project LICENSE file.
