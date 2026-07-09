package app

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"GithubReleaseNotificationAPI/services/subscription/internal/config"
)

func newGRPCServerTLSConfig(cfg *config.Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.GRPCTLSCert, cfg.GRPCTLSKey)
	if err != nil {
		return nil, fmt.Errorf("load grpc server cert: %w", err)
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
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
