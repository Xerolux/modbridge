package security

import (
	"errors"
	"net"
	"sync"
)

var (
	// ErrIPBlocked is returned when an IP is blocked.
	ErrIPBlocked = errors.New("IP address is blocked")
	// ErrFunctionBlocked is returned when a function code is blocked.
	ErrFunctionBlocked = errors.New("function code is blocked")
)

// IPFilter manages IP whitelisting and blacklisting.
type IPFilter struct {
	mu         sync.RWMutex
	whitelist  map[string]bool // IP -> allowed
	blacklist  map[string]bool // IP -> blocked
	mode       FilterMode
	networks   []*net.IPNet // CIDR ranges
}

// FilterMode defines the filtering mode.
type FilterMode int

const (
	// ModeWhitelist allows only whitelisted IPs.
	ModeWhitelist FilterMode = iota
	// ModeBlacklist blocks only blacklisted IPs.
	ModeBlacklist
)

// NewIPFilter creates a new IP filter.
func NewIPFilter(mode FilterMode) *IPFilter {
	return &IPFilter{
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
		mode:      mode,
		networks:  make([]*net.IPNet, 0),
	}
}

// AddWhitelist adds an IP to the whitelist.
func (f *IPFilter) AddWhitelist(ip string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if it's a CIDR range
	if _, network, err := net.ParseCIDR(ip); err == nil {
		f.networks = append(f.networks, network)
		return nil
	}

	// Single IP
	if net.ParseIP(ip) == nil {
		return errors.New("invalid IP address")
	}

	f.whitelist[ip] = true
	return nil
}

// AddBlacklist adds an IP to the blacklist.
func (f *IPFilter) AddBlacklist(ip string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if net.ParseIP(ip) == nil {
		return errors.New("invalid IP address")
	}

	f.blacklist[ip] = true
	return nil
}

// RemoveWhitelist removes an IP from the whitelist.
func (f *IPFilter) RemoveWhitelist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.whitelist, ip)
}

// RemoveBlacklist removes an IP from the blacklist.
func (f *IPFilter) RemoveBlacklist(ip string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.blacklist, ip)
}

// Allow checks if an IP is allowed.
func (f *IPFilter) Allow(ip string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check blacklist first
	if f.blacklist[ip] {
		return false
	}

	// Check mode
	switch f.mode {
	case ModeWhitelist:
		// Allow if in whitelist or in whitelisted network
		if f.whitelist[ip] {
			return true
		}

		for _, network := range f.networks {
			if network.Contains(parsedIP) {
				return true
			}
		}

		return false

	case ModeBlacklist:
		// Allow all except blacklisted
		return true

	default:
		return false
	}
}

// Clear clears all filters.
func (f *IPFilter) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.whitelist = make(map[string]bool)
	f.blacklist = make(map[string]bool)
	f.networks = make([]*net.IPNet, 0)
}

// FunctionCodeFilter filters Modbus function codes.
type FunctionCodeFilter struct {
	mu            sync.RWMutex
	allowedCodes  map[byte]bool
	blockedCodes  map[byte]bool
	mode          FilterMode
}

// NewFunctionCodeFilter creates a new function code filter.
func NewFunctionCodeFilter(mode FilterMode) *FunctionCodeFilter {
	return &FunctionCodeFilter{
		allowedCodes: make(map[byte]bool),
		blockedCodes: make(map[byte]bool),
		mode:         mode,
	}
}

// AddAllowed adds a function code to the allowed list.
func (f *FunctionCodeFilter) AddAllowed(code byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.allowedCodes[code] = true
}

// AddBlocked adds a function code to the blocked list.
func (f *FunctionCodeFilter) AddBlocked(code byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.blockedCodes[code] = true
}

// Allow checks if a function code is allowed.
func (f *FunctionCodeFilter) Allow(code byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check blocked list first
	if f.blockedCodes[code] {
		return false
	}

	switch f.mode {
	case ModeWhitelist:
		return f.allowedCodes[code]

	case ModeBlacklist:
		return true

	default:
		return false
	}
}

// SecurityPolicy combines multiple security filters.
type SecurityPolicy struct {
	ipFilter       *IPFilter
	functionFilter *FunctionCodeFilter
	enforceIP      bool
	enforceFunc    bool
}

// NewSecurityPolicy creates a new security policy.
func NewSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		ipFilter:       NewIPFilter(ModeBlacklist),
		functionFilter: NewFunctionCodeFilter(ModeBlacklist),
		enforceIP:      false,
		enforceFunc:    false,
	}
}

// SetIPFilter sets the IP filter and enables IP filtering.
func (p *SecurityPolicy) SetIPFilter(filter *IPFilter) {
	p.ipFilter = filter
	p.enforceIP = true
}

// SetFunctionCodeFilter sets the function code filter and enables filtering.
func (p *SecurityPolicy) SetFunctionCodeFilter(filter *FunctionCodeFilter) {
	p.functionFilter = filter
	p.enforceFunc = true
}

// Allow checks if a request is allowed based on IP and function code.
func (p *SecurityPolicy) Allow(ip string, functionCode byte) error {
	if p.enforceIP {
		if !p.ipFilter.Allow(ip) {
			return ErrIPBlocked
		}
	}

	if p.enforceFunc {
		if !p.functionFilter.Allow(functionCode) {
			return ErrFunctionBlocked
		}
	}

	return nil
}
