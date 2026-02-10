.PHONY: build build-embed run run-embed dev test test-e2e lint clean release-cli

build:
	go build -o bin/server .

build-embed: web-build
	go build -tags=embed -o bin/server-embed .

run:
	go run .

run-embed: build-embed
	./bin/server-embed

dev:
	trap 'kill 0' SIGINT; go run . & (cd web && npm run dev) & wait

test:
	go test -v -race ./...

test-e2e:
	go test -v -tags=e2e ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -rf bin/
	rm -rf internal/frontend/dist

# Frontend
.PHONY: web-install web-dev web-build

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build: web-install
	cd web && npm run build

# Release configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BINARY_NAME := mcp-registry

LDFLAGS := -s -w \
  -X 'github.com/mcp-reg/mcp-sub-registry/internal/version.Version=$(VERSION)' \
  -X 'github.com/mcp-reg/mcp-sub-registry/internal/version.GitCommit=$(GIT_COMMIT)' \
  -X 'github.com/mcp-reg/mcp-sub-registry/internal/version.BuildDate=$(BUILD_DATE)'

bin/$(BINARY_NAME)-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=embed -ldflags "$(LDFLAGS)" -o $@ .
	sha256sum $@ > $@.sha256

bin/$(BINARY_NAME)-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=embed -ldflags "$(LDFLAGS)" -o $@ .
	sha256sum $@ > $@.sha256

bin/$(BINARY_NAME)-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -tags=embed -ldflags "$(LDFLAGS)" -o $@ .
	sha256sum $@ > $@.sha256

bin/$(BINARY_NAME)-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags=embed -ldflags "$(LDFLAGS)" -o $@ .
	sha256sum $@ > $@.sha256

bin/$(BINARY_NAME)-windows-amd64.exe:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags=embed -ldflags "$(LDFLAGS)" -o $@ .
	sha256sum $@ > $@.sha256

release-cli: web-build bin/$(BINARY_NAME)-linux-amd64 \
             bin/$(BINARY_NAME)-linux-arm64 \
             bin/$(BINARY_NAME)-darwin-amd64 \
             bin/$(BINARY_NAME)-darwin-arm64 \
             bin/$(BINARY_NAME)-windows-amd64.exe
