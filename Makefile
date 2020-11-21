.PHONY: dev build install release test clean

CGO_ENABLED=0
VERSION=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "$VERSION")
COMMIT=$(shell git rev-parse --short HEAD || echo "$COMMIT")

all: dev

dev: build 
	@./sshbox -v

build:
	@go build -tags "netgo static_build" -installsuffix netgo \
		-ldflags "-w \
		-X $(shell go list).Version=$(VERSION) \
		-X $(shell go list).Commit=$(COMMIT)" \
		.

install: build
	@go install .

release:
	@./tools/release.sh

test:
	@go test -v -cover -race ./...

clean:
	@git clean -f -d -X
