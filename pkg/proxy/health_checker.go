package proxy

import (
	"context"
	"net"
	"sync"
	"time"
)

type HealthChecker struct {
	mu          sync.RWMutex
	target      string
	interval    time.Duration
	timeout     time.Duration
	healthy     bool
	consecutive int
	lastCheck   time.Time
	lastError   string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	log         func(string, string)
	onUnhealthy func()
	onRecovery  func()
	running     bool
	callbackMu  sync.Mutex // prevents overlapping callbacks
}

type HealthStatus struct {
	Healthy          bool      `json:"healthy"`
	LastCheck        time.Time `json:"last_check"`
	LastError        string    `json:"last_error"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	Interval         string    `json:"interval"`
}

func NewHealthChecker(target string, interval, timeout time.Duration, logFn func(string, string)) *HealthChecker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	hc := &HealthChecker{
		target:   target,
		interval: interval,
		timeout:  timeout,
		healthy:  true,
		log:      logFn,
		ctx:      ctx,
		cancel:   cancel,
	}

	return hc
}

func (hc *HealthChecker) Start() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.running {
		return
	}
	hc.running = true

	select {
	case <-hc.ctx.Done():
		hc.ctx, hc.cancel = context.WithCancel(context.Background())
	default:
	}

	hc.wg.Add(1)
	go hc.loop()
}

func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = false
	hc.mu.Unlock()

	hc.cancel()
	hc.wg.Wait()
}

func (hc *HealthChecker) loop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.check()
		}
	}
}

func (hc *HealthChecker) check() {
	ctx, cancel := context.WithTimeout(hc.ctx, hc.timeout)
	defer cancel()

	dialer := net.Dialer{Timeout: hc.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", hc.target)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastCheck = time.Now()

	if err != nil {
		wasHealthy := hc.healthy
		hc.consecutive++
		hc.healthy = false
		hc.lastError = err.Error()
		if hc.consecutive == 1 || hc.consecutive%5 == 0 {
			if hc.log != nil {
				hc.log("HC", "target unreachable: "+err.Error())
			}
		}
		if wasHealthy && hc.onUnhealthy != nil {
			hc.callbackMu.Lock()
			fn := hc.onUnhealthy
			hc.callbackMu.Unlock()
			if fn != nil {
				go fn()
			}
		}
		return
	}

	wasUnhealthy := !hc.healthy
	hc.consecutive = 0
	hc.healthy = true
	hc.lastError = ""
	conn.Close()

	if wasUnhealthy {
		if hc.log != nil {
			hc.log("HC", "target is reachable again")
		}
		hc.callbackMu.Lock()
		fn := hc.onRecovery
		hc.callbackMu.Unlock()
		if fn != nil {
			go fn()
		}
	}
}

func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.healthy
}

func (hc *HealthChecker) SetOnUnhealthy(fn func()) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onUnhealthy = fn
}

func (hc *HealthChecker) SetOnRecovery(fn func()) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onRecovery = fn
}

func (hc *HealthChecker) GetStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return HealthStatus{
		Healthy:          hc.healthy,
		LastCheck:        hc.lastCheck,
		LastError:        hc.lastError,
		ConsecutiveFails: hc.consecutive,
		Interval:         hc.interval.String(),
	}
}
