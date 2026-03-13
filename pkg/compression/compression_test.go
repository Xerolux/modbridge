package compression

import (
	"testing"
)

func TestNewCompressor(t *testing.T) {
	c := NewCompressor(CompressionGzip)
	if c == nil {
		t.Fatal("NewCompressor() returned nil")
	}
}

func TestCompressor_Compress(t *testing.T) {
	c := NewCompressor(CompressionGzip)

	data := []byte("Hello, World! This is a test string that should compress well.")

	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(compressed) >= len(data) {
		t.Logf("Warning: compressed size (%d) >= original size (%d)", len(compressed), len(data))
	}
}

func TestCompressor_Decompress(t *testing.T) {
	c := NewCompressor(CompressionGzip)

	original := []byte("Hello, World!")
	compressed, err := c.Compress(original)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	decompressed, err := c.Decompress(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	if string(decompressed) != string(original) {
		t.Errorf("Decompressed data doesn't match original")
	}
}

func TestCompressor_None(t *testing.T) {
	c := NewCompressor(CompressionNone)

	data := []byte("Hello, World!")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(compressed) != string(data) {
		t.Error("Expected data to remain unchanged with CompressionNone")
	}
}

func TestCompressor_ShouldCompress(t *testing.T) {
	c := NewCompressor(CompressionGzip)

	smallData := []byte("Hi")
	if c.ShouldCompress(smallData, 100) {
		t.Error("Expected small data not to compress (size < 100)")
	}

	largeData := make([]byte, 1000)
	for i := range largeData {
		largeData[i] = 'A'
	}
	if !c.ShouldCompress(largeData, 100) {
		t.Error("Expected large data to compress (size >= 100)")
	}
}

func TestStatsTracker(t *testing.T) {
	tracker := NewStatsTracker()

	tracker.Record(1000, 500)
	tracker.Record(500, 400)

	stats := tracker.Stats()
	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.TotalRequests)
	}
	if stats.TotalInputBytes != 1500 {
		t.Errorf("Expected 1500 input bytes, got %d", stats.TotalInputBytes)
	}
}
