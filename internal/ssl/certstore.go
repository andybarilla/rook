package ssl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type LocalCertStore struct {
	certsDir string
	caCert   *x509.Certificate
	caKey    *ecdsa.PrivateKey
}

func NewLocalCertStore(certsDir string) *LocalCertStore {
	return &LocalCertStore{certsDir: certsDir}
}

func (s *LocalCertStore) InstallCA() error {
	if err := os.MkdirAll(s.certsDir, 0o700); err != nil {
		return fmt.Errorf("create certs dir: %w", err)
	}

	caCertPath := filepath.Join(s.certsDir, "ca.pem")
	if _, err := os.Stat(caCertPath); err == nil {
		return s.loadCA()
	}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Flock Development CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA cert: %w", err)
	}

	caCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	if err := writePEM(caCertPath, "CERTIFICATE", certDER); err != nil {
		return fmt.Errorf("write CA cert: %w", err)
	}

	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	if err := writePEM(filepath.Join(s.certsDir, "ca-key.pem"), "EC PRIVATE KEY", caKeyDER); err != nil {
		return fmt.Errorf("write CA key: %w", err)
	}

	s.caCert = caCert
	s.caKey = caKey
	return nil
}

func (s *LocalCertStore) GenerateCert(domain string) error {
	if s.caCert == nil || s.caKey == nil {
		return fmt.Errorf("CA not initialized")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Flock"},
		},
		DNSNames:    []string{domain},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(825 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, s.caCert, &key.PublicKey, s.caKey)
	if err != nil {
		return fmt.Errorf("create cert: %w", err)
	}

	if err := writePEM(s.CertPath(domain), "CERTIFICATE", certDER); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	if err := writePEM(s.KeyPath(domain), "EC PRIVATE KEY", keyDER); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	return nil
}

func (s *LocalCertStore) CertPath(domain string) string {
	return filepath.Join(s.certsDir, domain+".pem")
}

func (s *LocalCertStore) KeyPath(domain string) string {
	return filepath.Join(s.certsDir, domain+"-key.pem")
}

func (s *LocalCertStore) HasCert(domain string) bool {
	_, certErr := os.Stat(s.CertPath(domain))
	_, keyErr := os.Stat(s.KeyPath(domain))
	return certErr == nil && keyErr == nil
}

func (s *LocalCertStore) loadCA() error {
	caCertPEM, err := os.ReadFile(filepath.Join(s.certsDir, "ca.pem"))
	if err != nil {
		return fmt.Errorf("read CA cert: %w", err)
	}
	block, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA cert: %w", err)
	}

	caKeyPEM, err := os.ReadFile(filepath.Join(s.certsDir, "ca-key.pem"))
	if err != nil {
		return fmt.Errorf("read CA key: %w", err)
	}
	keyBlock, _ := pem.Decode(caKeyPEM)
	caKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA key: %w", err)
	}

	s.caCert = caCert
	s.caKey = caKey
	return nil
}

func writePEM(path string, pemType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: pemType, Bytes: data})
}
