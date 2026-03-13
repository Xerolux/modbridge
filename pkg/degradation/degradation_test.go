package degradation

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestResourceManager_RegisterComponent(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)

	rm.RegisterComponent("test", ComponentImportant, 5, nil)

	if !rm.IsComponentEnabled("test") {
		t.Error("Expected component to be enabled")
	}

	status, ok := rm.GetComponentStatus("test")
	if !ok {
		t.Fatal("Component not found")
	}

	if status != StatusRunning {
		t.Errorf("Expected status Running, got %v", status)
	}
}

func TestResourceManager_CalculateLevel(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)
	rm.mu.Lock()
	rm.metrics = ResourceMetrics{
		MemoryPercent: 95,
		CPUPercent:    90,
	}
	rm.mu.Unlock()

	rm.mu.Lock()
	level := rm.calculateLevel()
	rm.mu.Unlock()

	if level != LevelCritical {
		t.Errorf("Expected Critical level, got %v", level)
	}
}

func TestResourceManager_DegradationLevels(t *testing.T) {
	tests := []struct {
		name     string
		metrics  ResourceMetrics
		expected Level
	}{
		{
			name: "No degradation",
			metrics: ResourceMetrics{
				MemoryPercent: 50,
				CPUPercent:    50,
			},
			expected: LevelNone,
		},
		{
			name: "Low degradation",
			metrics: ResourceMetrics{
				MemoryPercent: 75,
				CPUPercent:    70,
			},
			expected: LevelLow,
		},
		{
			name: "Medium degradation",
			metrics: ResourceMetrics{
				MemoryPercent: 85,
				CPUPercent:    80,
			},
			expected: LevelMedium,
		},
		{
			name: "Critical degradation",
			metrics: ResourceMetrics{
				MemoryPercent: 95,
				CPUPercent:    90,
			},
			expected: LevelCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewResourceManager(10*time.Second, 30*time.Second)
			rm.UpdateMetrics(tt.metrics)

			rm.evaluateResources()

			if rm.GetLevel() != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, rm.GetLevel())
			}
		})
	}
}

func TestResourceManager_ComponentDisabling(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)

	rm.RegisterComponent("critical", ComponentCritical, 1, nil)
	rm.RegisterComponent("important", ComponentImportant, 5, nil)
	rm.RegisterComponent("optional", ComponentOptional, 10, nil)

	rm.UpdateMetrics(ResourceMetrics{
		MemoryPercent: 75,
		CPUPercent:    75,
	})

	rm.evaluateResources()

	if rm.IsComponentEnabled("optional") {
		t.Error("Expected optional component to be disabled at low degradation")
	}

	if !rm.IsComponentEnabled("critical") {
		t.Error("Expected critical component to be enabled")
	}

	if !rm.IsComponentEnabled("important") {
		t.Error("Expected important component to be enabled")
	}
}

func TestResourceManager_MediumDegradation(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)

	rm.RegisterComponent("important", ComponentImportant, 5, nil)

	rm.UpdateMetrics(ResourceMetrics{
		MemoryPercent: 85,
		CPUPercent:    85,
	})

	rm.evaluateResources()

	status, _ := rm.GetComponentStatus("important")
	if status != StatusDegraded {
		t.Errorf("Expected Degraded status, got %v", status)
	}
}

func TestResourceManager_Recovery(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)

	rm.RegisterComponent("optional", ComponentOptional, 10, nil)

	rm.UpdateMetrics(ResourceMetrics{
		MemoryPercent: 75,
		CPUPercent:    75,
	})

	rm.evaluateResources()

	if rm.IsComponentEnabled("optional") {
		t.Error("Expected optional component to be disabled")
	}

	rm.UpdateMetrics(ResourceMetrics{
		MemoryPercent: 50,
		CPUPercent:    50,
	})

	rm.tryRecovery()

	if !rm.IsComponentEnabled("optional") {
		t.Error("Expected optional component to be re-enabled after recovery")
	}
}

func TestResourceManager_GetStats(t *testing.T) {
	rm := NewResourceManager(10*time.Second, 30*time.Second)

	rm.RegisterComponent("critical", ComponentCritical, 1, nil)
	rm.RegisterComponent("optional", ComponentOptional, 10, nil)

	rm.UpdateMetrics(ResourceMetrics{
		MemoryPercent: 75,
		CPUPercent:    75,
	})

	rm.evaluateResources()

	stats := rm.GetStats()

	if stats["level"] != LevelLow.String() {
		t.Errorf("Expected level 'low', got %v", stats["level"])
	}

	if stats["components_total"] != 2 {
		t.Errorf("Expected 2 total components, got %v", stats["components_total"])
	}
}

func TestResourceManager_ConcurrentAccess(t *testing.T) {
	rm := NewResourceManager(10*time.Millisecond, 50*time.Millisecond)

	rm.RegisterComponent("test", ComponentImportant, 5, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rm.Start(ctx)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rm.UpdateMetrics(ResourceMetrics{
				MemoryPercent: float64(50 + id*5),
				CPUPercent:    float64(50 + id*5),
			})
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rm.IsComponentEnabled("test")
			rm.GetComponentStatus("test")
			rm.GetLevel()
		}()
	}

	wg.Wait()
}
