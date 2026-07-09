package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	monconfig "GithubReleaseNotificationAPI/services/monitoring/internal/config"
)

func newGRPCClientTLSConfig(cfg *monconfig.Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.GRPCTLSCert, cfg.GRPCTLSKey)
	if err != nil {
		return nil, fmt.Errorf("load grpc client cert: %w", err)
	}

	caPEM, err := os.ReadFile(cfg.GRPCTLSCACert)
	if err != nil {
		return nil, fmt.Errorf("read grpc ca cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("parse grpc ca cert: invalid PEM")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
