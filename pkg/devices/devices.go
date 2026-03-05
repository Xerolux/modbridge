package devices

import (
	"log"
	"modbridge/pkg/database"
	"net"
	"sync"
	"time"
)

// Device represents a connected client device.
type Device struct {
	IP           string    `json:"ip"`
	MAC          string    `json:"mac"`
	Name         string    `json:"name"` // User-friendly name
	LastConnect  time.Time `json:"last_connect"`
	FirstSeen    time.Time `json:"first_seen"`
	RequestCount int64     `json:"request_count"`
	ProxyID      string    `json:"proxy_id"`
}

// dbWrite represents a database write operation.
type dbWrite struct {
	device  *database.Device
	ip      string
	proxyID string
	history bool // true if this is a connection history write
}

// Tracker tracks connected devices with persistent storage.
type Tracker struct {
	mu       sync.RWMutex
	devices  map[string]*Device // key: IP address (cache)
	db       *database.DB       // persistent storage
	dbWrites chan dbWrite       // channel for async database writes
	wg       sync.WaitGroup     // wait group for graceful shutdown
	stopOnce sync.Once          // ensures stop is called only once
	stopped  chan struct{}      // signals when writer is stopped
}

// NewTracker creates a new device tracker with database support.
func NewTracker(db *database.DB) *Tracker {
	t := &Tracker{
		devices:  make(map[string]*Device),
		db:       db,
		dbWrites: make(chan dbWrite, 5000), // Increased buffer size for high-load scenarios
		stopped:  make(chan struct{}),
	}

	// Load existing devices from database into cache
	if db != nil {
		t.loadDevicesFromDB()
		// Start database writer goroutine
		t.wg.Add(1)
		go t.dbWriter()
	}

	return t
}

// dbWriter processes database writes asynchronously.
func (t *Tracker) dbWriter() {
	defer t.wg.Done()
	for {
		select {
		case write, ok := <-t.dbWrites:
			if !ok {
				// Channel closed, drain complete, exit
				return
			}
			t.processWrite(write)
		case <-t.stopped:
			// Signal received to stop. Drain remaining buffered writes.
			for {
				select {
				case write, ok := <-t.dbWrites:
					if !ok {
						return
					}
					t.processWrite(write)
				default:
					return
				}
			}
		}
	}
}

// processWrite handles a single database write operation.
func (t *Tracker) processWrite(write dbWrite) {
	if write.history {
		if err := t.db.AddConnectionHistory(write.ip, write.proxyID); err != nil {
			log.Printf("[DeviceTracker] Failed to write connection history: %v", err)
		}
	} else if write.device != nil {
		if err := t.db.SaveDevice(write.device); err != nil {
			log.Printf("[DeviceTracker] Failed to save device: %v", err)
		}
	}
}

// Stop gracefully stops the tracker and flushes pending writes.
func (t *Tracker) Stop() {
	t.stopOnce.Do(func() {
		if t.db != nil {
			// Signal the writer to start draining
			close(t.stopped)
			// Wait for writer to finish draining
			t.wg.Wait()
		}
	})
}

// loadDevicesFromDB loads all devices from the database into the cache.
func (t *Tracker) loadDevicesFromDB() {
	dbDevices, err := t.db.GetAllDevices()
	if err != nil {
		// Log error but don't fail - can continue with empty cache
		return
	}

	for _, dbDev := range dbDevices {
		t.devices[dbDev.IP] = &Device{
			IP:           dbDev.IP,
			MAC:          dbDev.MAC,
			Name:         dbDev.Name,
			FirstSeen:    dbDev.FirstSeen,
			LastConnect:  dbDev.LastConnect,
			RequestCount: dbDev.RequestCount,
			ProxyID:      dbDev.ProxyID,
		}
	}
}

// TrackConnection records a connection from a client.
func (t *Tracker) TrackConnection(conn net.Conn, proxyID string) {
	if conn == nil {
		return
	}

	remoteAddr := conn.RemoteAddr()
	if remoteAddr == nil {
		return
	}

	ip, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		ip = remoteAddr.String()
	}

	mac := getMACAddress(ip)
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	device, exists := t.devices[ip]
	if !exists {
		device = &Device{
			IP:        ip,
			MAC:       mac,
			Name:      "", // Will be set by user
			FirstSeen: now,
			ProxyID:   proxyID,
		}
		t.devices[ip] = device
	}

	device.LastConnect = now
	device.RequestCount++
	device.ProxyID = proxyID

	// Queue database writes if available (non-blocking)
	if t.db != nil {
		dbDevice := &database.Device{
			IP:           device.IP,
			MAC:          device.MAC,
			Name:         device.Name,
			FirstSeen:    device.FirstSeen,
			LastConnect:  device.LastConnect,
			RequestCount: device.RequestCount,
			ProxyID:      device.ProxyID,
		}

		// Non-blocking send to channel
		select {
		case t.dbWrites <- dbWrite{device: dbDevice}:
			// Successfully queued
		default:
			// Channel full, log warning but don't block
			log.Printf("[DeviceTracker] Database write channel full, dropping device write for %s", ip)
		}

		// Also queue connection history
		select {
		case t.dbWrites <- dbWrite{ip: ip, proxyID: proxyID, history: true}:
			// Successfully queued
		default:
			log.Printf("[DeviceTracker] Database write channel full, dropping history write for %s", ip)
		}
	}
}

// GetDevices returns all tracked devices.
func (t *Tracker) GetDevices() []Device {
	t.mu.RLock()
	defer t.mu.RUnlock()

	devices := make([]Device, 0, len(t.devices))
	for _, d := range t.devices {
		devices = append(devices, *d)
	}
	return devices
}

// SetDeviceName sets a user-friendly name for a device.
func (t *Tracker) SetDeviceName(ip, name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	device, exists := t.devices[ip]
	if !exists {
		// Create device entry if it doesn't exist
		device = &Device{
			IP:        ip,
			Name:      name,
			FirstSeen: time.Now(),
		}
		t.devices[ip] = device
	} else {
		device.Name = name
	}

	// Save to database if available (non-blocking)
	if t.db != nil {
		dbDevice := &database.Device{
			IP:           device.IP,
			MAC:          device.MAC,
			Name:         device.Name,
			FirstSeen:    device.FirstSeen,
			LastConnect:  device.LastConnect,
			RequestCount: device.RequestCount,
			ProxyID:      device.ProxyID,
		}
		select {
		case t.dbWrites <- dbWrite{device: dbDevice}:
			// Successfully queued
		default:
			log.Printf("[DeviceTracker] Database write channel full, dropping name update for %s", ip)
		}
	}

	return nil
}

// GetConnectionHistory returns connection history for a device.
func (t *Tracker) GetConnectionHistory(ip string, limit int) ([]*database.ConnectionHistoryEntry, error) {
	if t.db == nil {
		return nil, nil
	}
	return t.db.GetConnectionHistory(ip, limit)
}

// GetAllConnectionHistory returns all connection history with optional proxy filter.
func (t *Tracker) GetAllConnectionHistory(proxyID string, limit int) ([]*database.ConnectionHistoryEntry, error) {
	if t.db == nil {
		return nil, nil
	}
	return t.db.GetAllConnectionHistory(proxyID, limit)
}

// getMACAddress attempts to get the MAC address for an IP.
// Note: This is limited and may not work in all scenarios.
func getMACAddress(ip string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "unknown"
	}

	// Check if it's localhost
	if parsedIP.IsLoopback() {
		return "localhost"
	}

	// Try to find MAC from ARP (limited, mainly works for local network)
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var netIP net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				netIP = v.IP
			case *net.IPAddr:
				netIP = v.IP
			}

			if netIP != nil && netIP.Equal(parsedIP) {
				return iface.HardwareAddr.String()
			}
		}
	}

	// If we can't find it, return unknown
	// In production, you might want to use ARP table parsing or
	// external tools for more accurate MAC resolution
	return "unknown"
}
