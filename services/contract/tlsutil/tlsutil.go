package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

func LoadCertPool(certPath, keyPath, caCertPath string) (tls.Certificate, *x509.CertPool, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("load cert: %w", err)
	}

	caPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("read ca cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return tls.Certificate{}, nil, errors.New("parse ca cert: invalid PEM")
	}

	return cert, caPool, nil
}
