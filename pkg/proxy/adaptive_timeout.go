package proxy

import (
	"sync"
	"time"
)

type AdaptiveTimeout struct {
	mu             sync.RWMutex
	baseRead       time.Duration
	baseConnect    time.Duration
	currentRead    time.Duration
	currentConnect time.Duration
	p95Latency     time.Duration
	samples        []time.Duration
	maxSamples     int
	scaleFactor    float64
}

func NewAdaptiveTimeout(baseRead, baseConnect time.Duration) *AdaptiveTimeout {
	return &AdaptiveTimeout{
		baseRead:       baseRead,
		baseConnect:    baseConnect,
		currentRead:    baseRead,
		currentConnect: baseConnect,
		maxSamples:     100,
		scaleFactor:    2.0,
	}
}

func (at *AdaptiveTimeout) Record(latency time.Duration) {
	at.mu.Lock()
	defer at.mu.Unlock()

	if len(at.samples) >= at.maxSamples {
		at.samples = at.samples[1:]
	}
	at.samples = append(at.samples, latency)

	if len(at.samples) >= 10 {
		sorted := make([]time.Duration, len(at.samples))
		copy(sorted, at.samples)
		sortDurations(sorted)
		p95Idx := len(sorted) * 95 / 100
		if p95Idx >= len(sorted) {
			p95Idx = len(sorted) - 1
		}
		at.p95Latency = sorted[p95Idx]

		adaptiveRead := time.Duration(float64(at.p95Latency) * at.scaleFactor)
		adaptiveConnect := time.Duration(float64(at.p95Latency) * at.scaleFactor)

		minRead := at.baseRead
		maxRead := at.baseRead * 10
		if adaptiveRead < minRead {
			adaptiveRead = minRead
		}
		if adaptiveRead > maxRead {
			adaptiveRead = maxRead
		}

		minConnect := at.baseConnect
		maxConnect := at.baseConnect * 10
		if adaptiveConnect < minConnect {
			adaptiveConnect = minConnect
		}
		if adaptiveConnect > maxConnect {
			adaptiveConnect = maxConnect
		}

		at.currentRead = adaptiveRead
		at.currentConnect = adaptiveConnect
	}
}

func (at *AdaptiveTimeout) GetReadTimeout() time.Duration {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.currentRead
}

func (at *AdaptiveTimeout) GetConnectTimeout() time.Duration {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.currentConnect
}

func (at *AdaptiveTimeout) GetP95() time.Duration {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.p95Latency
}

func sortDurations(d []time.Duration) {
	for i := 0; i < len(d); i++ {
		for j := i + 1; j < len(d); j++ {
			if d[j] < d[i] {
				d[i], d[j] = d[j], d[i]
			}
		}
	}
}
