package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

// CompressionType defines the compression algorithm
type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
)

// Compressor handles response compression
type Compressor struct {
	compressionType CompressionType
	level           int // Gzip compression level
	pool            sync.Pool
}

// NewCompressor creates a new compressor
func NewCompressor(compressionType CompressionType) *Compressor {
	c := &Compressor{
		compressionType: compressionType,
		level:           6, // Default gzip level
	}

	c.pool.New = func() interface{} {
		return new(bytes.Buffer)
	}

	return c
}

// Compress compresses data using the configured algorithm
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	switch c.compressionType {
	case CompressionGzip:
		return c.compressGzip(data)
	default:
		return data, nil
	}
}

// compressGzip compresses data using gzip
func (c *Compressor) compressGzip(data []byte) ([]byte, error) {
	buf := c.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer c.pool.Put(buf)

	writer, err := gzip.NewWriterLevel(buf, c.level)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompress decompresses gzip data
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ShouldCompress determines if data should be compressed
func (c *Compressor) ShouldCompress(data []byte, minSize int) bool {
	if c.compressionType == CompressionNone {
		return false
	}
	return len(data) >= minSize
}

// CompressionStats holds compression statistics
type CompressionStats struct {
	TotalRequests     uint64
	CompressedCount   uint64
	TotalInputBytes   uint64
	TotalOutputBytes  uint64
	CompressionRatio  float64
}

// StatsTracker tracks compression statistics
type StatsTracker struct {
	mu    sync.Mutex
	stats CompressionStats
}

// NewStatsTracker creates a new stats tracker
func NewStatsTracker() *StatsTracker {
	return &StatsTracker{}
}

// Record records a compression operation
func (t *StatsTracker) Record(inputSize, outputSize int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats.TotalRequests++
	t.stats.TotalInputBytes += uint64(inputSize)
	t.stats.TotalOutputBytes += uint64(outputSize)

	if outputSize < inputSize {
		t.stats.CompressedCount++
	}

	if t.stats.TotalInputBytes > 0 {
		t.stats.CompressionRatio = float64(t.stats.TotalInputBytes-t.stats.TotalOutputBytes) / float64(t.stats.TotalInputBytes)
	}
}

// Stats returns current statistics
func (t *StatsTracker) Stats() CompressionStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.stats
}
