# rook-ssl Plugin Design

## Goal

Provide automatic local TLS for Rook-managed sites using mkcert's Go library. Sites with `TLS: true` get trusted HTTPS certificates without manual setup.

## Architecture

Single `internal/ssl` package implementing `plugin.ServicePlugin` and `caddy.CertProvider`. A `CertStore` interface abstracts cert generation for testability.

### Interfaces

**`CertProvider`** (added to `internal/caddy/caddy.go`):

```go
type CertProvider interface {
    CertPair(domain string) (certFile, keyFile string, err error)
}
```

**`CertStore`** (in `internal/ssl/ssl.go`):

```go
type CertStore interface {
    InstallCA() error
    GenerateCert(domain string) error
    CertPath(domain string) string
    KeyPath(domain string) string
    HasCert(domain string) bool
}
```

### Components

**`ssl.Plugin`** — implements `plugin.ServicePlugin` + `caddy.CertProvider`:

- `Init(host)` — creates cert directory at `~/.local/share/rook/certs/`, initializes mkcert CA (installs to system trust store)
- `Start()` — iterates `host.Sites()`, generates certs for `TLS: true` sites missing certs
- `Stop()` — no-op (certs persist on disk)
- `CertPair(domain)` — returns `{certsDir}/{domain}.pem` and `{certsDir}/{domain}-key.pem`
- `ServiceStatus()` — Running if CA is trusted, Degraded otherwise

**`BuildConfig` enhancement** — signature changes to accept optional `CertProvider`. When `site.TLS == true` and cert is available, adds TLS connection policy with cert/key file paths to the Caddy JSON config.

### Data Flow

```
Site with TLS=true added
  → ssl.Plugin.Start() ensures cert exists via CertStore
  → Caddy BuildConfig(sites, resolver, certProvider)
  → For TLS sites: certProvider.CertPair(domain) → cert/key paths
  → Caddy JSON includes tls.certificates.load_files block
  → Caddy serves HTTPS with trusted local cert
```

### Error Handling

- CA install failure → plugin degraded, sites fall back to HTTP-only
- Cert generation failure for a domain → BuildConfig skips TLS for that site, logs warning
- Missing cert at runtime → CertPair returns error, Caddy serves HTTP

### Tech Stack

- Go 1.23
- `github.com/FiloSottile/mkcert` Go library (embedded, no external binary)
- Certs stored at `~/.local/share/rook/certs/`
- CA stored at mkcert's default CAROOT location
