# Getting Started

## 1. Start the application

If you installed the systemd service:
```bash
sudo systemctl start modbridge.service
```

Or manually:
```bash
./modbridge
```

## 2. First login

On first start, ModBridge automatically generates a secure admin password. You can find it in the console output or the systemd log.

```bash
# Find the password in the log
sudo journalctl -u modbridge.service -n 50 | grep "Initial admin password"
```

## 3. Open the web interface

Open a browser and navigate to:
`http://<your-server-ip>:8080`

Log in with:
* **Username:** admin
* **Password:** (the generated password from step 2)

After the first login you will be prompted to change the password.

## 4. Create a proxy

1. Go to **Proxies** in the left-hand menu
2. Click **New Proxy**
3. Fill in the details:
   * **Name:** e.g. "Inverter 1"
   * **Local port:** e.g. `:5020` (the port ModBridge should listen on)
   * **Target address:** e.g. `192.168.1.100:502` (the IP address of your Modbus device)
4. Click **Save**
5. Enable the proxy using the toggle in the list

## 5. Connect a Modbus client

Configure your SCADA system, Home Assistant, or Node-RED to connect to ModBridge instead of directly to the device:

* **IP:** `<IP-of-the-ModBridge-server>`
* **Port:** `5020` (or the local port configured in step 4)

That's it! ModBridge now forwards the requests and logs the traffic.
