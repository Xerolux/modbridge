package proxy

import (
	"context"
	"net"
	"sync"
	"time"
)

// DeadConnectionDetector monitors and detects dead/unhealthy connections
type DeadConnectionDetector struct {
	mu                sync.RWMutex
	connections       map[string]*ConnectionInfo
	checkInterval     time.Duration
	maxIdleTime       time.Duration
	maxErrors         int
	muStats           sync.Mutex
	detectedCount     int64
	recoveredCount    int64
	lastCheck         time.Time
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	running           bool
}

// ConnectionInfo holds information about a connection
type ConnectionInfo struct {
	Conn           net.Conn
	RemoteAddr     string
	LastActivity   time.Time
	ErrorCount     int
	LastError      error
	LastErrorTime  time.Time
	IsDead         bool
	CreatedAt      time.Time
	BytesRead      int64
	BytesWritten   int64
}

// DetectorConfig holds configuration for the detector
type DetectorConfig struct {
	CheckInterval   time.Duration // How often to check connections (default: 10s)
	MaxIdleTime     time.Duration // Max idle time before marking suspicious (default: 60s)
	MaxErrors       int           // Max errors before marking dead (default: 5)
}

// DefaultDetectorConfig returns sensible defaults
func DefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		CheckInterval: 10 * time.Second,
		MaxIdleTime:   60 * time.Second,
		MaxErrors:     5,
	}
}

// NewDeadConnectionDetector creates a new detector
func NewDeadConnectionDetector(config DetectorConfig) *DeadConnectionDetector {
	if config.CheckInterval <= 0 {
		config.CheckInterval = 10 * time.Second
	}
	if config.MaxIdleTime <= 0 {
		config.MaxIdleTime = 60 * time.Second
	}
	if config.MaxErrors <= 0 {
		config.MaxErrors = 5
	}

	ctx, cancel := context.WithCancel(context.Background())

	detector := &DeadConnectionDetector{
		connections:   make(map[string]*ConnectionInfo),
		checkInterval: config.CheckInterval,
		maxIdleTime:   config.MaxIdleTime,
		maxErrors:     config.MaxErrors,
		ctx:           ctx,
		cancel:        cancel,
		running:       true,
		lastCheck:     time.Now(),
	}

	// Start detection loop
	detector.wg.Add(1)
	go detector.detectionLoop()

	return detector
}

// RegisterConnection registers a connection for monitoring
func (d *DeadConnectionDetector) RegisterConnection(conn net.Conn, remoteAddr string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.connections[remoteAddr] = &ConnectionInfo{
		Conn:         conn,
		RemoteAddr:   remoteAddr,
		LastActivity: time.Now(),
		CreatedAt:    time.Now(),
		IsDead:       false,
	}
}

// UnregisterConnection removes a connection from monitoring
func (d *DeadConnectionDetector) UnregisterConnection(remoteAddr string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.connections, remoteAddr)
}

// RecordActivity records activity on a connection
func (d *DeadConnectionDetector) RecordActivity(remoteAddr string, bytesRead, bytesWritten int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if connInfo, exists := d.connections[remoteAddr]; exists {
		connInfo.LastActivity = time.Now()
		connInfo.BytesRead += int64(bytesRead)
		connInfo.BytesWritten += int64(bytesWritten)
		connInfo.ErrorCount = 0 // Reset error count on successful activity
		connInfo.IsDead = false
	}
}

// RecordError records an error on a connection
func (d *DeadConnectionDetector) RecordError(remoteAddr string, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if connInfo, exists := d.connections[remoteAddr]; exists {
		connInfo.ErrorCount++
		connInfo.LastError = err
		connInfo.LastErrorTime = time.Now()

		// Check if connection should be marked as dead
		if connInfo.ErrorCount >= d.maxErrors {
			connInfo.IsDead = true

			d.muStats.Lock()
			d.detectedCount++
			d.muStats.Unlock()
		}
	}
}

// IsDead checks if a connection is marked as dead
func (d *DeadConnectionDetector) IsDead(remoteAddr string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if connInfo, exists := d.connections[remoteAddr]; exists {
		return connInfo.IsDead
	}
	return false
}

// detectionLoop periodically checks all connections
func (d *DeadConnectionDetector) detectionLoop() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.checkConnections()
		}
	}
}

// checkConnections checks all registered connections
func (d *DeadConnectionDetector) checkConnections() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastCheck = time.Now()
	now := time.Now()

	for _, connInfo := range d.connections {
		// Skip if already marked as dead
		if connInfo.IsDead {
			continue
		}

		// Check idle time
		idleTime := now.Sub(connInfo.LastActivity)
		if idleTime > d.maxIdleTime {
			// Check if connection is actually still alive
			if !d.isConnectionAlive(connInfo.Conn) {
				connInfo.IsDead = true

				d.muStats.Lock()
				d.detectedCount++
				d.muStats.Unlock()
			}
		}
	}
}

// isConnectionAlive checks if a connection is still alive
func (d *DeadConnectionDetector) isConnectionAlive(conn net.Conn) bool {
	// Set a very short deadline to test the connection
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	defer conn.SetReadDeadline(time.Time{}) // Reset deadline

	// Try to read 0 bytes (just a peek)
	one := make([]byte, 1)
	_, err := conn.Read(one)

	// If we get EOF, connection is closed
	if err == net.ErrClosed || err.Error() == "EOF" {
		return false
	}

	// Timeout means no data but connection is still there
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Any other error means connection is dead
	return err == nil
}

// GetDeadConnections returns all dead connections
func (d *DeadConnectionDetector) GetDeadConnections() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var dead []string
	for addr, connInfo := range d.connections {
		if connInfo.IsDead {
			dead = append(dead, addr)
		}
	}

	return dead
}

// GetConnectionInfo returns info about a specific connection
func (d *DeadConnectionDetector) GetConnectionInfo(remoteAddr string) (*ConnectionInfo, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	connInfo, exists := d.connections[remoteAddr]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	copy := *connInfo
	return &copy, true
}

// GetStats returns detector statistics
func (d *DeadConnectionDetector) GetStats() map[string]interface{} {
	d.muStats.Lock()
	defer d.muStats.Unlock()

	d.mu.RLock()
	totalConnections := len(d.connections)
	deadConnections := 0
	for _, connInfo := range d.connections {
		if connInfo.IsDead {
			deadConnections++
		}
	}
	d.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":  totalConnections,
		"dead_connections":   deadConnections,
		"detected_count":     d.detectedCount,
		"recovered_count":    d.recoveredCount,
		"last_check":         d.lastCheck,
		"check_interval":     d.checkInterval,
	}
}

// Stop stops the detector
func (d *DeadConnectionDetector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return
	}

	d.running = false
	d.cancel()
	d.wg.Wait()
}

// Recover marks a connection as recovered
func (d *DeadConnectionDetector) Recover(remoteAddr string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if connInfo, exists := d.connections[remoteAddr]; exists {
		connInfo.IsDead = false
		connInfo.ErrorCount = 0
		connInfo.LastActivity = time.Now()

		d.muStats.Lock()
		d.recoveredCount++
		d.muStats.Unlock()
	}
}
