package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// NewTLSConfig create a new TLS configuration
func NewTLSConfig(tlsCertCaFile, tlsCertFile, tlsCertKeyFile string, insecureSkipVerify bool) (*tls.Config, error) {
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(tlsCertCaFile)
	if err != nil {
		return nil, fmt.Errorf("load CA certificate for %s: %w", tlsCertFile, err)
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, fmt.Errorf("append CA certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:            rootCertPool,
		InsecureSkipVerify: insecureSkipVerify, // Skip hostname verification for IP addresses
	}

	// support mTLS if cert and key files are provided
	if tlsCertFile != "" && tlsCertKeyFile != "" {
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(tlsCertFile, tlsCertKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate(%s) and key(%s): %w", tlsCertFile, tlsCertKeyFile, err)
		}
		clientCert = append(clientCert, certs)

		tlsConfig.Certificates = clientCert
	}

	return tlsConfig, nil
}
