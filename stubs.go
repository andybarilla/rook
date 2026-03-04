package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// loggingCaddyRunner logs Caddy operations without running real Caddy.
type loggingCaddyRunner struct {
	logger *log.Logger
}

func (r *loggingCaddyRunner) Run(cfgJSON []byte) error {
	r.logger.Printf("caddy: loading config (%d bytes)", len(cfgJSON))
	return nil
}

func (r *loggingCaddyRunner) Stop() error {
	r.logger.Println("caddy: stopped")
	return nil
}

// loggingFPMRunner logs FPM operations without running real php-fpm.
type loggingFPMRunner struct {
	logger *log.Logger
}

func (r *loggingFPMRunner) StartPool(version string) error {
	r.logger.Printf("php-fpm: starting pool for PHP %s", version)
	return nil
}

func (r *loggingFPMRunner) StopPool(version string) error {
	r.logger.Printf("php-fpm: stopping pool for PHP %s", version)
	return nil
}

func (r *loggingFPMRunner) PoolSocket(version string) string {
	return fmt.Sprintf("/tmp/php-fpm-%s.sock", version)
}

// loggingCertStore logs cert operations without generating real certs.
type loggingCertStore struct {
	logger   *log.Logger
	certsDir string
}

func (s *loggingCertStore) InstallCA() error {
	s.logger.Println("ssl: CA install (stub)")
	return nil
}

func (s *loggingCertStore) GenerateCert(domain string) error {
	s.logger.Printf("ssl: generating cert for %s (stub)", domain)
	os.MkdirAll(s.certsDir, 0o755)
	// Create empty placeholder files so HasCert returns true
	os.WriteFile(s.CertPath(domain), []byte("stub"), 0o644)
	os.WriteFile(s.KeyPath(domain), []byte("stub"), 0o644)
	return nil
}

func (s *loggingCertStore) CertPath(domain string) string {
	return filepath.Join(s.certsDir, domain+".pem")
}

func (s *loggingCertStore) KeyPath(domain string) string {
	return filepath.Join(s.certsDir, domain+"-key.pem")
}

func (s *loggingCertStore) HasCert(domain string) bool {
	_, err := os.Stat(s.CertPath(domain))
	return err == nil
}
