package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Config holds TLS configuration
type Config struct {
	// Certificate file path
	CertFile string `json:"cert_file" yaml:"cert_file"`
	// Private key file path
	KeyFile string `json:"key_file" yaml:"key_file"`
	// CA certificate file path (for client cert verification)
	CAFile string `json:"ca_file" yaml:"ca_file"`
	// Minimum TLS version
	MinVersion string `json:"min_version" yaml:"min_version"`
	// Maximum TLS version
	MaxVersion string `json:"max_version" yaml:"max_version"`
	// Cipher suites (comma-separated)
	CipherSuites string `json:"cipher_suites" yaml:"cipher_suites"`
	// Client authentication mode
	ClientAuth string `json:"client_auth" yaml:"client_auth"` // none, request, require, verify-ca, verify-cert
	// Enable session resumption
	SessionTickets bool `json:"session_tickets" yaml:"session_tickets"`
	// Client CA file for mTLS
	ClientCAFile string `json:"client_ca_file" yaml:"client_ca_file"`
}

// DefaultConfig returns default TLS configuration
func DefaultConfig() *Config {
	return &Config{
		CertFile:       "certs/server.crt",
		KeyFile:        "certs/server.key",
		CAFile:         "certs/ca.crt",
		MinVersion:     "1.2",
		MaxVersion:     "1.3",
		ClientAuth:     "none",
		SessionTickets: true,
	}
}

// Manager manages TLS certificates and configuration
type Manager struct {
	config    *Config
	cert      *tls.Certificate
	certPool  *x509.CertPool
	clientCAs *x509.CertPool
}

// NewManager creates a new TLS manager
func NewManager(cfg *Config) (*Manager, error) {
	m := &Manager{
		config: cfg,
	}

	// Load certificates
	if err := m.LoadCertificates(); err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	return m, nil
}

// LoadCertificates loads TLS certificates from files
func (m *Manager) LoadCertificates() error {
	// Load server certificate
	if m.config.CertFile != "" && m.config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(m.config.CertFile, m.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load key pair: %w", err)
		}
		m.cert = &cert
	}

	// Load CA certificate for client verification
	if m.config.CAFile != "" {
		caCert, err := os.ReadFile(m.config.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		m.certPool = x509.NewCertPool()
		if !m.certPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}
	}

	// Load client CA for mTLS
	if m.config.ClientCAFile != "" {
		clientCA, err := os.ReadFile(m.config.ClientCAFile)
		if err != nil {
			return fmt.Errorf("failed to read client CA certificate: %w", err)
		}

		m.clientCAs = x509.NewCertPool()
		if !m.clientCAs.AppendCertsFromPEM(clientCA) {
			return fmt.Errorf("failed to parse client CA certificate")
		}
	}

	return nil
}

// GetTLSConfig creates a tls.Config from the manager
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	cfg := &tls.Config{
		Certificates: []tls.Certificate{*m.cert},
		MinVersion:   m.parseTLSVersion(m.config.MinVersion),
		MaxVersion:   m.parseTLSVersion(m.config.MaxVersion),
	}

	// Set cipher suites if specified
	if m.config.CipherSuites != "" {
		cfg.CipherSuites = m.parseCipherSuites(m.config.CipherSuites)
	} else {
		// Use secure defaults
		cfg.CipherSuites = m.getDefaultCipherSuites()
	}

	// Set client authentication mode
	cfg.ClientAuth = m.parseClientAuth(m.config.ClientAuth)

	// Set CA pool for client verification
	if m.certPool != nil {
		cfg.RootCAs = m.certPool
	}

	// Set client CA pool for mTLS
	if m.clientCAs != nil {
		cfg.ClientCAs = m.clientCAs
	}

	// Enable session tickets
	cfg.SessionTicketsDisabled = !m.config.SessionTickets

	// Set preferred server cipher suites
	cfg.PreferServerCipherSuites = true

	return cfg, nil
}

// parseTLSVersion converts string version to tls version constant
func (m *Manager) parseTLSVersion(version string) uint16 {
	switch version {
	case "1.0":
		return tls.VersionTLS10
	case "1.1":
		return tls.VersionTLS11
	case "1.2":
		return tls.VersionTLS12
	case "1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Default to 1.2
	}
}

// parseCipherSuites parses cipher suite string to IDs
func (m *Manager) parseCipherSuites(suites string) []uint16 {
	// Map of cipher suite names to IDs
	suiteMap := map[string]uint16{
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_AES_128_GCM_SHA256":                  tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                  tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":            tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	var ids []uint16
	for _, suite := range splitAndTrim(suites, ",") {
		if id, ok := suiteMap[suite]; ok {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return m.getDefaultCipherSuites()
	}

	return ids
}

// getDefaultCipherSuites returns secure default cipher suites
func (m *Manager) getDefaultCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}
}

// parseClientAuth parses client authentication mode
func (m *Manager) parseClientAuth(mode string) tls.ClientAuthType {
	switch mode {
	case "none":
		return tls.NoClientCert
	case "request":
		return tls.RequestClientCert
	case "require":
		return tls.RequireAnyClientCert
	case "verify-ca":
		return tls.VerifyClientCertIfGiven
	case "verify-cert":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.NoClientCert
	}
}

// Reload reloads certificates without restarting
func (m *Manager) Reload() error {
	return m.LoadCertificates()
}

// GenerateSelfSignedCert generates a self-signed certificate for testing
func GenerateSelfSignedCert(host string, validFor time.Duration) (certPEM, keyPEM []byte, err error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ModBridge"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validFor),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{host},
		IPAddresses:           []net.IP{},
	}

	// Add localhost and common IP addresses
	template.DNSNames = append(template.DNSNames, "localhost")
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("::1"))

	// Generate certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	// Encode private key to PEM
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	return certPEM, keyPEM, nil
}

// SaveCertificates saves certificate and key to files
func SaveCertificates(certDir, host string, validFor time.Duration) error {
	// Create directory
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Generate certificate
	certPEM, keyPEM, err := GenerateSelfSignedCert(host, validFor)
	if err != nil {
		return err
	}

	// Save certificate
	certPath := filepath.Join(certDir, "server.crt")
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyPath := filepath.Join(certDir, "server.key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// CreateTLSListener creates a TLS listener
func CreateTLSListener(addr string, cfg *Config) (*Listener, error) {
	tlsConfig, err := cfg.ToTLSConfig()
	if err != nil {
		return nil, err
	}

	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener: %w", err)
	}

	return &Listener{listener}, nil
}

// Listener wraps a TLS listener
type Listener struct {
	listener net.Listener
}

// ToTLSConfig converts Config to tls.Config
func (cfg *Config) ToTLSConfig() (*tls.Config, error) {
	mgr, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}

	return mgr.GetTLSConfig()
}

// GetCertificateInfo returns information about the certificate
func (m *Manager) GetCertificateInfo() (*CertInfo, error) {
	if m.cert == nil || len(m.cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate loaded")
	}

	x509Cert, err := x509.ParseCertificate(m.cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	info := &CertInfo{
		Subject:            x509Cert.Subject.CommonName,
		Issuer:             x509Cert.Issuer.CommonName,
		NotBefore:          x509Cert.NotBefore,
		NotAfter:           x509Cert.NotAfter,
		DNSNames:           x509Cert.DNSNames,
		IPAddresses:        x509Cert.IPAddresses,
		IsSelfSigned:       x509Cert.IsCA,
		SerialNumber:       x509Cert.SerialNumber.String(),
		SignatureAlgorithm: x509Cert.SignatureAlgorithm.String(),
	}

	return info, nil
}

// CertInfo holds certificate information
type CertInfo struct {
	Subject            string    `json:"subject"`
	Issuer             string    `json:"issuer"`
	NotBefore          time.Time `json:"not_before"`
	NotAfter           time.Time `json:"not_after"`
	DNSNames           []string  `json:"dns_names"`
	IPAddresses        []net.IP  `json:"ip_addresses"`
	IsSelfSigned       bool      `json:"is_self_signed"`
	SerialNumber       string    `json:"serial_number"`
	SignatureAlgorithm string    `json:"signature_algorithm"`
}

// IsValid checks if the certificate is currently valid
func (ci *CertInfo) IsValid() bool {
	now := time.Now()
	return now.After(ci.NotBefore) && now.Before(ci.NotAfter)
}

// DaysUntilExpiry returns the number of days until the certificate expires
func (ci *CertInfo) DaysUntilExpiry() int {
	return int(ci.NotAfter.Sub(time.Now()).Hours() / 24)
}

// splitAndTrim splits a string and trims whitespace
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by separator
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var parts []string
	current := ""

	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, current)
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}

	parts = append(parts, current)
	return parts
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

// parseIP parses an IP address string
func parseIP(s string) []byte {
	// Simple implementation - in production use net.ParseIP
	// This is a placeholder to avoid import issues
	return []byte(s)
}

// MutualTLSConfig provides mutual TLS configuration
func MutualTLSConfig(certFile, keyFile, caFile string) (*Config, error) {
	config := &Config{
		CertFile:   certFile,
		KeyFile:    keyFile,
		ClientAuth: "verify-cert",
		CAFile:     caFile,
	}
	return config, nil
}

// VerifyClientCertificate verifies client certificate
func (m *Manager) VerifyClientCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// TODO: Implement custom certificate verification
	return nil
}
