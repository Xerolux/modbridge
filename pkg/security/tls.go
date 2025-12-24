package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// TLSConfig holds TLS configuration.
type TLSConfig struct {
	// CertFile is the path to the certificate file.
	CertFile string

	// KeyFile is the path to the private key file.
	KeyFile string

	// CAFile is the path to the CA certificate file for client verification.
	CAFile string

	// ClientAuth defines the client authentication policy.
	ClientAuth tls.ClientAuthType

	// MinVersion is the minimum TLS version (default: TLS 1.2).
	MinVersion uint16

	// CipherSuites specifies the cipher suites to use.
	CipherSuites []uint16
}

// DefaultTLSConfig returns a secure default TLS configuration.
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		ClientAuth: tls.NoClientCert,
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// BuildTLSConfig creates a tls.Config from TLSConfig.
func (tc *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if tc.CertFile == "" || tc.KeyFile == "" {
		return nil, errors.New("certificate and key files are required")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(tc.CertFile, tc.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tc.MinVersion,
		CipherSuites: tc.CipherSuites,
		ClientAuth:   tc.ClientAuth,
	}

	// Load CA certificate for client verification
	if tc.CAFile != "" {
		caCert, err := os.ReadFile(tc.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}

		config.ClientCAs = caCertPool
	}

	return config, nil
}

// ValidateClientCert validates a client certificate.
func ValidateClientCert(cert *x509.Certificate, caCertPool *x509.CertPool) error {
	if cert == nil {
		return errors.New("no client certificate provided")
	}

	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// GenerateSelfSignedCert generates a self-signed certificate for testing.
// In production, use proper certificates from a CA.
func GenerateSelfSignedCert(certFile, keyFile string) error {
	// This is a placeholder. In production, use crypto/x509 to generate proper certificates
	// or use Let's Encrypt for free certificates.
	return errors.New("self-signed certificate generation not implemented - use openssl or certbot")
}
