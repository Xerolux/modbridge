# Modbus Proxy Manager

A modern, robust Modbus TCP proxy manager with an "Apple-Style" web interface.

## Features

- **Multi-Proxy Support**: Manage multiple Modbus TCP proxy instances.
- **Web Interface**: Clean, responsive UI built with Vue.js and Tailwind CSS (embedded).
- **Live Observability**: Real-time traffic logging via Server-Sent Events (SSE) and status monitoring.
- **Security**: 
  - Admin authentication (Bcrypt hashing).
  - Secure session cookies.
  - Read-only views for unauthenticated users (optional).
- **Configuration**: JSON-based persistence with Export/Import capabilities.
- **Single Binary**: The entire frontend is embedded in the Go binary.

## Quick Start

1.  **Download** the `modbusmanager` binary (or build it).
2.  **Run** the application:
    ```bash
    ./modbusmanager
    ```
3.  **Open Browser**: Go to `http://localhost:8080`.
4.  **First Run Setup**: You will be prompted to set an Admin Password.
5.  **Login**: Use the password to access the full management interface.

## Usage

### Dashboard
View the overall health of your proxies, including total request counts and error rates.

### Proxies
- **Add Proxy**: Click "+ Add Proxy" to define a new route.
    - **Name**: A friendly name.
    - **Listen Address**: The local port to listen on (e.g., `:5020`).
    - **Target Address**: The downstream Modbus device (e.g., `192.168.1.50:502`).
- **Control**: Start, Stop, or Restart proxies individually.

### Logs
- **Live View**: Watch logs stream in real-time.
- **Filter**: Filter by text or log level (INFO/WARN/ERROR).
- **Download**: Export the recent logs to a JSON file.

### Configuration
- **Export**: Download the current configuration as `config.json`.
- **Import**: Restore a configuration file. Note: This overwrites current settings (except the admin password).

## Building from Source

Requirements: Go 1.20+

```bash
# Clone repository
git clone https://github.com/yourusername/modbus-proxy-manager.git
cd modbus-proxy-manager

# Build
go build -o modbusmanager main.go

# Run
./modbusmanager
```

## Configuration File

The configuration is stored in `config.json` in the working directory.

```json
{
  "web_port": ":8080",
  "admin_pass_hash": "$2a$14$...",
  "proxies": [
    {
      "id": "uuid...",
      "name": "Solar Inverter",
      "listen_addr": ":5020",
      "target_addr": "192.168.1.100:502",
      "enabled": true
    }
  ]
}
```

## License

MIT
