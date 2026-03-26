.PHONY: build build-cli build-gui install install-cli install-gui test clean dev-gui deps-gui

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/andybarilla/rook/internal/cli.Version=$(VERSION)"

# Build both binaries
build: build-cli build-gui

# Build CLI only
build-cli:
	go build $(LDFLAGS) -o bin/rook ./cmd/rook

# Build GUI (requires frontend build first)
build-gui: deps-gui
	cd cmd/rook-gui/frontend && npx vite build
	go build $(LDFLAGS) -tags "production,webkit2_41" -o bin/rook-gui ./cmd/rook-gui

# Install both to $GOPATH/bin
install: install-cli install-gui

# Install CLI to $GOPATH/bin
install-cli:
	go install $(LDFLAGS) ./cmd/rook

# Install GUI to $GOPATH/bin (requires frontend build first)
install-gui: deps-gui
	cd cmd/rook-gui/frontend && npx vite build
	go install $(LDFLAGS) -tags "production,webkit2_41" ./cmd/rook-gui

# Install frontend dependencies
deps-gui:
	cd cmd/rook-gui/frontend && npm install

# Run GUI in dev mode (hot reload)
dev-gui: deps-gui
	cd cmd/rook-gui && wails dev

# Run all tests
test:
	go test ./... -timeout 30s

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf cmd/rook-gui/frontend/dist
	rm -rf cmd/rook-gui/frontend/node_modules
