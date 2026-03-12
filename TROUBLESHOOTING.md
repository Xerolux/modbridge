# Troubleshooting Guide

## Table of Contents
- [Installation Issues](#installation-issues)
- [Startup Problems](#startup-problems)
- [Connection Issues](#connection-issues)
- [Performance Issues](#performance-issues)
- [Authentication Issues](#authentication-issues)
- [Database Issues](#database-issues)
- [Docker Issues](#docker-issues)

---

## Installation Issues

### Problem: Go dependencies fail to download
**Symptoms:**
```
go: github.com/go-ldap/ldap/v3@v3.4.6: verifying go.mod...
```

**Solution:**
```bash
# Clean Go module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify
go mod verify
```

### Problem: Build fails with "cannot find package"
**Symptoms:**
```
package modbridge/pkg/rbac is not in GOROOT
```

**Solution:**
```bash
# Ensure you're in the modbridge directory
cd /path/to/modbridge

# Run go mod tidy
go mod tidy

# Build again
go build -o modbridge ./main.go
```

---

## Startup Problems

### Problem: Port 8080 already in use
**Symptoms:**
```
ERROR: Failed to start server: bind: address already in use
```

**Solution:**
```bash
# Find process using the port
lsof -i :8080  # Linux/macOS
netstat -ano | findstr :8080  # Windows

# Either kill the process or use a different port
WEB_PORT=:9090 ./modbridge
```

### Problem: Database initialization fails
**Symptoms:**
```
ERROR: Failed to initialize schema: table devices already exists
```

**Solution:**
```bash
# Backup existing database
cp modbridge.db modbridge.db.backup

# Remove and restart
rm modbridge.db
./modbridge
```

---

## Connection Issues

### Problem: Cannot connect to Modbus device
**Symptoms:**
- Proxy shows "Disconnected" status
- Timeout errors in logs

**Diagnosis:**
```bash
# Test network connectivity
ping 192.168.1.100

# Test Modbus port
nc -zv 192.168.1.100 502

# Check firewall
sudo iptables -L -n | grep 502
```

**Solution:**
1. Verify device is powered on and reachable
2. Check Modbus port (usually 502)
3. Verify target address in proxy configuration
4. Increase timeout values in proxy config
5. Check device documentation for Modbus settings

### Problem: High latency on Modbus requests
**Symptoms:**
- Requests take >100ms
- Timeout errors

**Solution:**
```json
{
  "connection_timeout": 15,
  "read_timeout": 15,
  "max_retries": 1
}
```

Enable register caching to reduce device load.

---

## Performance Issues

### Problem: High memory usage
**Symptoms:**
- Process using >500MB RAM
- Memory grows over time

**Diagnosis:**
```bash
# Check memory usage
ps aux | grep modbridge
pmap $(pidof modbridge)

# Check for memory leaks
curl http://localhost:9090/metrics | grep heap
```

**Solution:**
1. Enable log rotation:
```json
{
  "log_max_size": 100,
  "log_max_files": 10,
  "log_max_age_days": 30
}
```

2. Reduce cache retention
3. Check for connection leaks
4. Restart service periodically if needed

### Problem: High CPU usage
**Symptoms:**
- CPU constantly >50%
- Slow response times

**Diagnosis:**
```bash
# Profile CPU usage
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof
```

**Solution:**
1. Check for infinite loops in custom code
2. Reduce polling frequency
3. Enable connection pooling
4. Check for excessive logging

---

## Authentication Issues

### Problem: Cannot login after password change
**Symptoms:**
- "Invalid credentials" error
- Locked out of system

**Solution:**
```bash
# Stop modbridge
sudo systemctl stop modbridge

# Reset admin password (edit config.json)
# Set "admin_pass_hash" to empty string

# Restart
sudo systemctl start modbridge

# Check logs for new temporary password
sudo journalctl -u modbridge -n 50
```

### Problem: Session expires too quickly
**Symptoms:**
- Frequent re-login required
- "Session expired" errors

**Solution:**
```json
{
  "session_timeout": 168  // 7 days in hours
}
```

---

## Database Issues

### Problem: Database is locked
**Symptoms:**
```
database is locked (5)
```

**Solution:**
```bash
# Check for other processes
lsof modbridge.db

# Enable WAL mode (should be automatic)
sqlite3 modbridge.db "PRAGMA journal_mode=WAL;"

# Check for corruption
sqlite3 modbridge.db "PRAGMA integrity_check;"
```

### Problem: Database grows too large
**Symptoms:**
- modbridge.db > 1GB
- Slow queries

**Solution:**
```bash
# Vacuum database
sqlite3 modbridge.db "VACUUM;"

# Clean old history
sqlite3 modbridge.db "DELETE FROM connection_history WHERE connected_at < datetime('now', '-30 days');"

# Analyze and reindex
sqlite3 modbridge.db "ANALYZE;"
sqlite3 modbridge.db "REINDEX;"
```

---

## Docker Issues

### Problem: Container exits immediately
**Symptoms:**
```
docker ps -a
STATUS: Exited (1) X seconds ago
```

**Diagnosis:**
```bash
# Check logs
docker logs modbridge

# Inspect container
docker inspect modbridge
```

**Common Causes & Solutions:**

1. **Port conflict:**
```bash
docker run -p 9090:8080 modbridge  # Use different host port
```

2. **Missing config file:**
```bash
docker run -v $(pwd)/config.json:/app/config.json modbridge
```

3. **Permission issues:**
```bash
docker run -u $(id -u):$(id -g) modbridge
```

### Problem: Cannot access Web UI from Docker
**Symptoms:**
- Connection refused
- Timeout

**Solution:**
```bash
# Ensure port mapping is correct
docker run -p 8080:8080 modbridge

# Check firewall
sudo firewall-cmd --add-port=8080/tcp --permanent

# Test locally first
docker exec -it modbridge wget http://localhost:8080
```

---

## Known Issues & Limitations

### Issue #1: Max 1000 concurrent connections
**Impact:** High-traffic deployments may hit connection limit

**Workaround:**
```json
{
  "max_connections": 5000  // Increase limit
}
```

**Status:** Will be addressed in v2.0 with connection pooling improvements

### Issue #2: Modbus RTU not supported
**Impact:** Cannot connect to serial/RTU devices

**Workaround:** Use a serial-to-TCP converter or gateway

**Status:** Planned for v2.1

### Issue #3: No automatic failover
**Impact:** Single point of failure

**Workaround:** Use external load balancer with health checks

**Status:** High availability mode in development for v2.0

---

## Getting Help

If you're still experiencing issues:

1. **Check logs:**
```bash
journalctl -u modbridge -f  # systemd
docker logs -f modbridge     # Docker
tail -f proxy.log            # Direct run
```

2. **Enable debug mode:**
```bash
DEBUG=true ./modbridge
```

3. **Collect diagnostics:**
```bash
curl http://localhost:8080/api/system/diagnostics > diagnostics.json
```

4. **Report issues:**
   - GitHub: https://github.com/Xerolux/modbridge/issues
   - Include: ModBridge version, OS, logs, diagnostics

5. **Community support:**
   - Discussions: https://github.com/Xerolux/modbridge/discussions
   - Documentation: https://github.com/Xerolux/modbridge/wiki

---

## Quick Reference: Common Commands

```bash
# Restart service
sudo systemctl restart modbridge

# View logs
sudo journalctl -u modbridge -f

# Test API
curl http://localhost:8080/api/health

# Check ports
netstat -tulpn | grep 8080

# Backup database
cp modbridge.db modbridge.db.$(date +%Y%m%d)

# Export config
curl http://localhost:8080/api/config/export > config-backup.json

# Restore config
curl -X PUT http://localhost:8080/api/config/import -H "Content-Type: application/json" -d @config-backup.json
```
