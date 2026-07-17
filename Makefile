build:
	go vet   ./...
	go build ./cmd/goto

install:
	go install -ldflags=-s ./cmd/goto

.PHONY: build install
