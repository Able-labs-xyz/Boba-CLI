VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w \
	-X github.com/tradeboba/boba-cli/internal/version.Version=$(VERSION) \
	-X github.com/tradeboba/boba-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/tradeboba/boba-cli/internal/version.Date=$(DATE)

.PHONY: build install clean test lint release

build:
	go build -ldflags "$(LDFLAGS)" -o bin/boba ./cmd/boba

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/boba

clean:
	rm -rf bin/

test:
	go test ./...

lint:
	golangci-lint run ./...

release:
	goreleaser release --clean
