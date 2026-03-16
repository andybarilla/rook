.PHONY: build build-cli build-gui install install-cli install-gui test clean dev-gui deps-gui

# Build both binaries
build: build-cli build-gui

# Build CLI only
build-cli:
	go build -o bin/rook ./cmd/rook

# Build GUI (requires frontend build first)
build-gui: deps-gui
	cd cmd/rook-gui/frontend && npx vite build
	go build -tags production -o bin/rook-gui ./cmd/rook-gui

# Install both to $GOPATH/bin
install: install-cli install-gui

# Install CLI to $GOPATH/bin
install-cli:
	go install ./cmd/rook

# Install GUI to $GOPATH/bin (requires frontend build first)
install-gui: deps-gui
	cd cmd/rook-gui/frontend && npx vite build
	go install -tags production ./cmd/rook-gui

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
