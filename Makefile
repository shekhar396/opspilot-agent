VERSION ?= dev
COMMIT ?= unknown
BUILD_DATE ?= unknown
GOOS ?= linux
GOARCH ?= amd64

MODULE := github.com/shekhar396/opspilot-agent
LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.Commit=$(COMMIT) -X $(MODULE)/internal/version.Date=$(BUILD_DATE)

.PHONY: help fmt test test-race vet build build-release check clean install uninstall package

help:
	@echo "Available targets:"
	@echo "  fmt            Format Go source"
	@echo "  test           Run unit tests"
	@echo "  test-race      Run tests with the race detector"
	@echo "  vet            Run go vet"
	@echo "  build          Build the development binary"
	@echo "  build-release  Build a static release binary in dist/"
	@echo "  check          Verify formatting, tests, race tests, vet, and build"
	@echo "  clean          Remove bin/ and dist/"
	@echo "  install        Install the current binary and Linux service (run as root)"
	@echo "  uninstall      Uninstall while preserving config and state (run as root)"
	@echo "  package        Build amd64/arm64 Linux archives and checksums"

fmt:
	gofmt -w cmd internal

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build -o bin/opspilot-agent ./cmd/opspilot-agent

build-release:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags '$(LDFLAGS)' -o dist/opspilot-agent ./cmd/opspilot-agent

check:
	test -z "$$(gofmt -l cmd internal)"
	go test ./...
	go test -race ./...
	go vet ./...
	go build -o bin/opspilot-agent ./cmd/opspilot-agent

clean:
	rm -rf -- bin dist

install: build
	./scripts/install.sh

uninstall:
	./scripts/uninstall.sh

package:
	VERSION='$(VERSION)' COMMIT='$(COMMIT)' BUILD_DATE='$(BUILD_DATE)' ./scripts/package.sh
