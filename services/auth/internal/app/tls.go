package app

import (
	"crypto/tls"
	"fmt"

	"GithubReleaseNotificationAPI/contract/tlsutil"
	"GithubReleaseNotificationAPI/services/auth/internal/config"
)

func newGRPCServerTLSConfig(cfg *config.Config) (*tls.Config, error) {
	cert, caPool, err := tlsutil.LoadCertPool(cfg.GRPCTLSCert, cfg.GRPCTLSKey, cfg.GRPCTLSCACert)
	if err != nil {
		return nil, fmt.Errorf("grpc server tls: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
