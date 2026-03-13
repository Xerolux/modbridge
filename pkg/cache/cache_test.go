package cache

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(nil)
	if cache == nil {
		t.Fatal("NewCache() returned nil")
	}
	if cache.config == nil {
		t.Error("cache config is nil")
	}
	cache.Close()
}

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(nil)
	defer cache.Close()

	cache.Set("key1", "value1", 0)

	value, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(nil)
	defer cache.Close()

	cache.Set("key1", "value1", 10*time.Millisecond)

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	_, ok = cache.Get("key1")
	if ok {
		t.Error("Expected key1 to be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	cache := NewCache(nil)
	defer cache.Close()

	cache.Set("key1", "value1", 0)

	_, ok := cache.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}

	cache.Delete("key1")

	_, ok = cache.Get("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(nil)
	defer cache.Close()

	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)

	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCache_LRU(t *testing.T) {
	config := &CacheConfig{
		MaxSize:         3,
		DefaultTTL:      0,
		CleanupInterval: 1 * time.Minute,
	}
	cache := NewCache(config)
	defer cache.Close()

	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	// Access key1 to make it more recent
	cache.Get("key1")

	// Add key4, should evict key2 (least recently used)
	cache.Set("key4", "value4", 0)

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	// key2 should be evicted
	_, ok := cache.Get("key2")
	if ok {
		t.Error("Expected key2 to be evicted")
	}

	// key1, key3, key4 should still exist
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Expected key1 to exist")
	}
	if _, ok := cache.Get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}
	if _, ok := cache.Get("key4"); !ok {
		t.Error("Expected key4 to exist")
	}
}

func TestCache_Stats(t *testing.T) {
	cache := NewCache(nil)
	defer cache.Close()

	cache.Set("key1", "value1", 0)
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestRegisterCache(t *testing.T) {
	cache := NewRegisterCache(nil)
	defer cache.cache.Close()

	values := []uint16{100, 200, 300}
	cache.SetRegister("device1", 0, values, 0)

	retrieved, ok := cache.GetRegister("device1", 0, 3)
	if !ok {
		t.Error("Expected to find cached registers")
	}

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 values, got %d", len(retrieved))
	}

	if retrieved[0] != 100 || retrieved[1] != 200 || retrieved[2] != 300 {
		t.Errorf("Unexpected values: %v", retrieved)
	}
}
