package examples

import (
	"context"
	"fmt"
	"log"
	"modbusproxy/pkg/plugin"
	"net"
	"time"
)

// LoggerMiddleware is an example middleware plugin that logs all requests/responses.
type LoggerMiddleware struct {
	verbose bool
}

// NewLoggerMiddleware creates a new logger middleware plugin.
func NewLoggerMiddleware(verbose bool) *plugin.Plugin {
	middleware := &LoggerMiddleware{
		verbose: verbose,
	}

	return &plugin.Plugin{
		Name:        "logger",
		Version:     "1.0.0",
		Description: "Logs all requests and responses",
		Author:      "Modbridge Team",
		Middleware:  middleware,
		Metadata: map[string]interface{}{
			"verbose": verbose,
		},
		OnLoad: func() error {
			log.Println("[Plugin] Logger middleware loaded")
			return nil
		},
		OnUnload: func() error {
			log.Println("[Plugin] Logger middleware unloaded")
			return nil
		},
	}
}

// Name returns the middleware name.
func (l *LoggerMiddleware) Name() string {
	return "logger"
}

// Priority returns the middleware priority.
func (l *LoggerMiddleware) Priority() int {
	return 10 // Run after other middlewares
}

// ProcessRequest processes an incoming request.
func (l *LoggerMiddleware) ProcessRequest(ctx context.Context, req []byte, conn net.Conn) ([]byte, error) {
	if l.verbose {
		log.Printf("[Logger] Request from %s: %d bytes", conn.RemoteAddr(), len(req))
		log.Printf("[Logger] Request data: %X", req)
	} else {
		log.Printf("[Logger] Request from %s: %d bytes", conn.RemoteAddr(), len(req))
	}

	return req, nil
}

// ProcessResponse processes an outgoing response.
func (l *LoggerMiddleware) ProcessResponse(ctx context.Context, resp []byte, conn net.Conn) ([]byte, error) {
	if l.verbose {
		log.Printf("[Logger] Response to %s: %d bytes", conn.RemoteAddr(), len(resp))
		log.Printf("[Logger] Response data: %X", resp)
	} else {
		log.Printf("[Logger] Response to %s: %d bytes", conn.RemoteAddr(), len(resp))
	}

	return resp, nil
}

// CacheMiddleware is an example middleware plugin that caches responses.
type CacheMiddleware struct {
	cache map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	response  []byte
	timestamp time.Time
}

// NewCacheMiddleware creates a new cache middleware plugin.
func NewCacheMiddleware(ttl time.Duration) *plugin.Plugin {
	middleware := &CacheMiddleware{
		cache: make(map[string]cacheEntry),
		ttl:   ttl,
	}

	return &plugin.Plugin{
		Name:        "cache",
		Version:     "1.0.0",
		Description: "Caches responses for faster access",
		Author:      "Modbridge Team",
		Middleware:  middleware,
		Metadata: map[string]interface{}{
			"ttl": ttl.String(),
		},
		OnLoad: func() error {
			log.Println("[Plugin] Cache middleware loaded")
			return nil
		},
		OnUnload: func() error {
			log.Println("[Plugin] Cache middleware unloaded")
			return nil
		},
	}
}

// Name returns the middleware name.
func (c *CacheMiddleware) Name() string {
	return "cache"
}

// Priority returns the middleware priority.
func (c *CacheMiddleware) Priority() int {
	return 5 // Run before other middlewares
}

// ProcessRequest processes an incoming request.
func (c *CacheMiddleware) ProcessRequest(ctx context.Context, req []byte, conn net.Conn) ([]byte, error) {
	// Check cache
	key := fmt.Sprintf("%X", req)
	if entry, exists := c.cache[key]; exists {
		if time.Since(entry.timestamp) < c.ttl {
			log.Printf("[Cache] Cache hit for request: %d bytes", len(req))
			// Store cached response in context for ProcessResponse to use
			// This is a simplified example
		}
	}

	return req, nil
}

// ProcessResponse processes an outgoing response.
func (c *CacheMiddleware) ProcessResponse(ctx context.Context, resp []byte, conn net.Conn) ([]byte, error) {
	// Cache the response (simplified, should use request as key)
	// In a real implementation, you'd extract the request from context
	log.Printf("[Cache] Caching response: %d bytes", len(resp))

	return resp, nil
}
