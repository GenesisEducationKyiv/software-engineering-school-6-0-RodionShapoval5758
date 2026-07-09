package main

import (
	"crypto/tls"
	"fmt"

	"GithubReleaseNotificationAPI/contract/tlsutil"
	monconfig "GithubReleaseNotificationAPI/services/monitoring/internal/config"
)

func newGRPCClientTLSConfig(cfg *monconfig.Config) (*tls.Config, error) {
	cert, caPool, err := tlsutil.LoadCertPool(cfg.GRPCTLSCert, cfg.GRPCTLSKey, cfg.GRPCTLSCACert)
	if err != nil {
		return nil, fmt.Errorf("grpc client tls: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
