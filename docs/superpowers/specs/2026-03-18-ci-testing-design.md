# Design: GitHub Actions CI for Testing

**Date:** 2026-03-18
**Status:** Approved

## Problem

No CI exists. Tests only run locally. PRs can be merged without passing tests.

## Decision

Add a single GitHub Actions workflow that runs all tests (including Docker-dependent ones) on PRs to main.

## Scope

### Created

1. **`.github/workflows/test.yml`** — Single workflow file with one job:
   - **Trigger:** Pull requests targeting `main`
   - **Runner:** `ubuntu-latest` (has Docker pre-installed)
   - **Steps:**
     1. `actions/checkout@v4`
     2. `actions/setup-go@v5` with Go 1.22
     3. `go test ./... -timeout 60s`
     4. `go vet ./...`

### Design Decisions

- **Single job, no matrix:** One Go version (1.22), one OS. No need for complexity.
- **60s timeout:** Bumped from Makefile's 30s to give Docker tests headroom in CI where containers may start slower.
- **No caching:** Only dependency is `gopkg.in/yaml.v3`. Not worth the config overhead.
- **No GUI build:** GUI requires webkit2gtk and npm — not needed for test CI.
- **`go vet` included:** Free static analysis, catches common mistakes.

### Not Included

- Branch protection rules (user can configure manually in GitHub settings)
- GUI build/test job
- Linting (golangci-lint etc.)
- Release/deploy workflows
