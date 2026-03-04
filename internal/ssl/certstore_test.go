package ssl_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/ssl"
)

func TestLocalCertStoreInstallCA(t *testing.T) {
	dir := t.TempDir()
	store := ssl.NewLocalCertStore(dir)

	if err := store.InstallCA(); err != nil {
		t.Fatalf("InstallCA: %v", err)
	}

	// CA files should exist
	caCertPath := filepath.Join(dir, "ca.pem")
	if _, err := os.Stat(caCertPath); err != nil {
		t.Fatal("ca.pem not found")
	}
	if _, err := os.Stat(filepath.Join(dir, "ca-key.pem")); err != nil {
		t.Fatal("ca-key.pem not found")
	}

	// CA cert should be a valid CA
	caCertPEM, _ := os.ReadFile(caCertPath)
	block, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	if !caCert.IsCA {
		t.Error("CA cert should have IsCA=true")
	}
}

func TestLocalCertStoreInstallCAIdempotent(t *testing.T) {
	dir := t.TempDir()
	store := ssl.NewLocalCertStore(dir)

	_ = store.InstallCA()
	// Second call should load existing CA, not error
	if err := store.InstallCA(); err != nil {
		t.Fatalf("second InstallCA: %v", err)
	}
}

func TestLocalCertStoreGenerateCert(t *testing.T) {
	dir := t.TempDir()
	store := ssl.NewLocalCertStore(dir)

	if err := store.InstallCA(); err != nil {
		t.Fatalf("InstallCA: %v", err)
	}
	if err := store.GenerateCert("app.test"); err != nil {
		t.Fatalf("GenerateCert: %v", err)
	}

	if !store.HasCert("app.test") {
		t.Error("HasCert should be true after generation")
	}
	if store.CertPath("app.test") != filepath.Join(dir, "app.test.pem") {
		t.Errorf("CertPath = %q, want %q", store.CertPath("app.test"), filepath.Join(dir, "app.test.pem"))
	}
	if store.KeyPath("app.test") != filepath.Join(dir, "app.test-key.pem") {
		t.Errorf("KeyPath = %q, want %q", store.KeyPath("app.test"), filepath.Join(dir, "app.test-key.pem"))
	}

	// Cert should have correct SAN
	certPEM, _ := os.ReadFile(store.CertPath("app.test"))
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	if len(cert.DNSNames) != 1 || cert.DNSNames[0] != "app.test" {
		t.Errorf("DNSNames = %v, want [app.test]", cert.DNSNames)
	}

	// Cert should be signed by CA
	caCertPEM, _ := os.ReadFile(filepath.Join(dir, "ca.pem"))
	caBlock, _ := pem.Decode(caCertPEM)
	caCert, _ := x509.ParseCertificate(caBlock.Bytes)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Errorf("cert not signed by CA: %v", err)
	}
}

func TestLocalCertStoreGenerateCertWithoutCA(t *testing.T) {
	dir := t.TempDir()
	store := ssl.NewLocalCertStore(dir)

	err := store.GenerateCert("app.test")
	if err == nil {
		t.Error("expected error generating cert without CA")
	}
}

func TestLocalCertStoreHasCertFalseWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store := ssl.NewLocalCertStore(dir)

	if store.HasCert("missing.test") {
		t.Error("HasCert should be false for missing domain")
	}
}
