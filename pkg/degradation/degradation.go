package degradation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ComponentType represents the type of component
type ComponentType int

const (
	// ComponentCritical are components that must always be available
	ComponentCritical ComponentType = iota
	// ComponentImportant are important but can be temporarily degraded
	ComponentImportant
	// ComponentOptional are optional components that can be disabled
	ComponentOptional
)

// ComponentStatus represents the current status of a component
type ComponentStatus int

const (
	// StatusRunning means component is running at full capacity
	StatusRunning ComponentStatus = iota
	// StatusDegraded means component is running in degraded mode
	StatusDegraded
	// StatusDisabled means component is disabled
	StatusDisabled
)

// Component represents a service component that can be degraded
type Component struct {
	Name         string
	Type         ComponentType
	Status       ComponentStatus
	Priority     int
	Enabled      bool
	DisableCount int
	LastDisabled time.Time
	Dependencies []string
}

// Level represents the degradation level
type Level int

const (
	// LevelNone means no degradation (all components running)
	LevelNone Level = iota
	// LevelLow means optional components are disabled
	LevelLow
	// LevelMedium means important components are degraded
	LevelMedium
	// LevelCritical means only critical components running
	LevelCritical
)

// ResourceManager manages system resources for degradation decisions
type ResourceManager struct {
	mu                  sync.RWMutex
	level               Level
	components          map[string]*Component
	metrics             ResourceMetrics
	thresholds          ResourceThresholds
	checkInterval       time.Duration
	recoveryInterval    time.Duration
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	onDegradationChange []func(Level, Level)
}

// ResourceMetrics represents current resource usage
type ResourceMetrics struct {
	MemoryPercent     float64
	CPUPercent        float64
	GoroutineCount    int
	ActiveConnections int
	DiskUsagePercent  float64
	OpenFiles         int
}

// ResourceThresholds defines thresholds for degradation levels
type ResourceThresholds struct {
	// Low degradation thresholds
	LowMemoryPercent      float64
	LowCPUPercent         float64
	LowGoroutineCount     int
	LowDiskUsagePercent   float64

	// Medium degradation thresholds
	MediumMemoryPercent    float64
	MediumCPUPercent       float64
	MediumGoroutineCount   int
	MediumDiskUsagePercent float64

	// Critical degradation thresholds
	CriticalMemoryPercent    float64
	CriticalCPUPercent       float64
	CriticalGoroutineCount   int
	CriticalDiskUsagePercent float64
}

// DefaultThresholds returns sensible default thresholds
func DefaultThresholds() ResourceThresholds {
	return ResourceThresholds{
		LowMemoryPercent:      70,
		LowCPUPercent:         70,
		LowGoroutineCount:     1000,
		LowDiskUsagePercent:   80,

		MediumMemoryPercent:    80,
		MediumCPUPercent:       85,
		MediumGoroutineCount:   5000,
		MediumDiskUsagePercent: 85,

		CriticalMemoryPercent:    90,
		CriticalCPUPercent:       95,
		CriticalGoroutineCount:   10000,
		CriticalDiskUsagePercent: 95,
	}
}

// NewResourceManager creates a new resource manager
func NewResourceManager(checkInterval, recoveryInterval time.Duration) *ResourceManager {
	return &ResourceManager{
		level:            LevelNone,
		components:       make(map[string]*Component),
		thresholds:       DefaultThresholds(),
		checkInterval:    checkInterval,
		recoveryInterval: recoveryInterval,
		stopCh:           make(chan struct{}),
	}
}

// RegisterComponent registers a component for degradation management
func (rm *ResourceManager) RegisterComponent(name string, ctype ComponentType, priority int, dependencies []string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.components[name] = &Component{
		Name:         name,
		Type:         ctype,
		Status:       StatusRunning,
		Priority:     priority,
		Enabled:      true,
		Dependencies: dependencies,
	}
}

// SetThresholds sets custom resource thresholds
func (rm *ResourceManager) SetThresholds(thresholds ResourceThresholds) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.thresholds = thresholds
}

// UpdateMetrics updates the current resource metrics
func (rm *ResourceManager) UpdateMetrics(metrics ResourceMetrics) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.metrics = metrics
}

// GetLevel returns the current degradation level
func (rm *ResourceManager) GetLevel() Level {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.level
}

// IsComponentEnabled checks if a component is currently enabled
func (rm *ResourceManager) IsComponentEnabled(name string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if comp, ok := rm.components[name]; ok {
		return comp.Enabled && comp.Status != StatusDisabled
	}
	return false
}

// GetComponentStatus returns the status of a component
func (rm *ResourceManager) GetComponentStatus(name string) (ComponentStatus, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if comp, ok := rm.components[name]; ok {
		return comp.Status, true
	}
	return StatusDisabled, false
}

// OnDegradationChange registers a callback for degradation level changes
func (rm *ResourceManager) OnDegradationChange(fn func(Level, Level)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onDegradationChange = append(rm.onDegradationChange, fn)
}

// Start begins the resource monitoring loop
func (rm *ResourceManager) Start(ctx context.Context) {
	rm.wg.Add(1)
	go rm.monitor(ctx)
}

// Stop stops the resource monitoring
func (rm *ResourceManager) Stop() {
	close(rm.stopCh)
	rm.wg.Wait()
}

// monitor monitors resources and adjusts degradation level
func (rm *ResourceManager) monitor(ctx context.Context) {
	defer rm.wg.Done()

	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()

	recoveryTicker := time.NewTicker(rm.recoveryInterval)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.evaluateResources()
		case <-recoveryTicker.C:
			rm.tryRecovery()
		case <-rm.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// evaluateResources evaluates current resources and adjusts degradation level
func (rm *ResourceManager) evaluateResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	newLevel := rm.calculateLevel()

	if newLevel != rm.level {
		oldLevel := rm.level
		rm.level = newLevel
		rm.applyDegradation(newLevel)
		rm.notifyLevelChange(oldLevel, newLevel)
	}
}

// calculateLevel calculates the appropriate degradation level based on metrics
func (rm *ResourceManager) calculateLevel() Level {
	m := rm.metrics
	t := rm.thresholds

	// Check critical thresholds first
	if m.MemoryPercent >= t.CriticalMemoryPercent ||
		m.CPUPercent >= t.CriticalCPUPercent ||
		m.GoroutineCount >= t.CriticalGoroutineCount ||
		m.DiskUsagePercent >= t.CriticalDiskUsagePercent {
		return LevelCritical
	}

	// Check medium thresholds
	if m.MemoryPercent >= t.MediumMemoryPercent ||
		m.CPUPercent >= t.MediumCPUPercent ||
		m.GoroutineCount >= t.MediumGoroutineCount ||
		m.DiskUsagePercent >= t.MediumDiskUsagePercent {
		return LevelMedium
	}

	// Check low thresholds
	if m.MemoryPercent >= t.LowMemoryPercent ||
		m.CPUPercent >= t.LowCPUPercent ||
		m.GoroutineCount >= t.LowGoroutineCount ||
		m.DiskUsagePercent >= t.LowDiskUsagePercent {
		return LevelLow
	}

	return LevelNone
}

// applyDegradation applies the degradation level to components
func (rm *ResourceManager) applyDegradation(level Level) {
	for _, comp := range rm.components {
		shouldEnable := rm.shouldEnableComponent(comp, level)

		if shouldEnable != comp.Enabled {
			comp.Enabled = shouldEnable
			if !shouldEnable {
				comp.Status = StatusDisabled
				comp.DisableCount++
				comp.LastDisabled = time.Now()
			} else {
				comp.Status = StatusRunning
			}
		}

		// Update degraded status for important components
		if comp.Type == ComponentImportant && level == LevelMedium && comp.Enabled {
			comp.Status = StatusDegraded
		}
	}
}

// shouldEnableComponent determines if a component should be enabled at the given level
func (rm *ResourceManager) shouldEnableComponent(comp *Component, level Level) bool {
	switch comp.Type {
	case ComponentCritical:
		// Critical components always enabled unless dependencies fail
		return true
	case ComponentImportant:
		// Important components enabled at low and medium degradation
		return level <= LevelMedium
	case ComponentOptional:
		// Optional components only enabled when no degradation
		return level == LevelNone
	}
	return false
}

// tryRecovery attempts to recover from degraded state
func (rm *ResourceManager) tryRecovery() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Only try recovery if we're currently degraded
	if rm.level == LevelNone {
		return
	}

	// Re-evaluate to see if we can recover
	newLevel := rm.calculateLevel()
	if newLevel < rm.level {
		oldLevel := rm.level
		rm.level = newLevel
		rm.applyDegradation(newLevel)
		rm.notifyLevelChange(oldLevel, newLevel)
	}
}

// notifyLevelChange notifies callbacks of degradation level change
func (rm *ResourceManager) notifyLevelChange(oldLevel, newLevel Level) {
	for _, fn := range rm.onDegradationChange {
		go fn(oldLevel, newLevel)
	}
}

// GetStats returns statistics about the degradation state
func (rm *ResourceManager) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := map[string]interface{}{
		"level":            rm.level.String(),
		"components_total": len(rm.components),
		"components_enabled": 0,
		"components_disabled": 0,
		"metrics": map[string]interface{}{
			"memory_percent":     rm.metrics.MemoryPercent,
			"cpu_percent":        rm.metrics.CPUPercent,
			"goroutine_count":    rm.metrics.GoroutineCount,
			"disk_usage_percent": rm.metrics.DiskUsagePercent,
		},
	}

	for _, comp := range rm.components {
		if comp.Enabled {
			stats["components_enabled"] = stats["components_enabled"].(int) + 1
		} else {
			stats["components_disabled"] = stats["components_disabled"].(int) + 1
		}
	}

	return stats
}

// String returns the string representation of the degradation level
func (l Level) String() string {
	switch l {
	case LevelNone:
		return "none"
	case LevelLow:
		return "low"
	case LevelMedium:
		return "medium"
	case LevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// String returns the string representation of component status
func (s ComponentStatus) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusDegraded:
		return "degraded"
	case StatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// Helper function to wrap component execution with degradation check
func (rm *ResourceManager) ExecuteIfEnabled(name string, fn func() error) error {
	if !rm.IsComponentEnabled(name) {
		return fmt.Errorf("component %s is disabled due to resource degradation", name)
	}
	return fn()
}

// ExecuteWithDegradation executes a function with optional degraded fallback
func (rm *ResourceManager) ExecuteWithDegradation(name string, normalFn, degradedFn func() error) error {
	status, ok := rm.GetComponentStatus(name)
	if !ok || status == StatusDisabled {
		return fmt.Errorf("component %s is disabled", name)
	}

	if status == StatusDegraded && degradedFn != nil {
		return degradedFn()
	}

	return normalFn()
}

// ManualDegradation allows manual degradation level setting
func (rm *ResourceManager) ManualDegradation(level Level) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if level < LevelNone || level > LevelCritical {
		return fmt.Errorf("invalid degradation level: %d", level)
	}

	oldLevel := rm.level
	rm.level = level
	rm.applyDegradation(level)
	rm.notifyLevelChange(oldLevel, level)

	return nil
}

// GetDisabledComponents returns a list of disabled components
func (rm *ResourceManager) GetDisabledComponents() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var disabled []string
	for name, comp := range rm.components {
		if !comp.Enabled {
			disabled = append(disabled, name)
		}
	}
	return disabled
}
