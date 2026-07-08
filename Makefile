build:
	go vet   ./...
	go build ./cmd/goto

.PHONY: build
