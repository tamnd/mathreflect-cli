# Build into bin/ (gitignored) so the binary never collides with the mathreflect/
# source package at the repo root.
BINARY  := bin/mathreflect
PKG     := ./cmd/mathreflect
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/tamnd/mathreflect-cli/cli.Version=$(VERSION) \
	-X github.com/tamnd/mathreflect-cli/cli.Commit=$(COMMIT) \
	-X github.com/tamnd/mathreflect-cli/cli.Date=$(DATE)

.PHONY: build install test vet fmt clean run

build:
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w -s .

clean:
	rm -rf bin dist

run: build
	./$(BINARY) $(ARGS)
