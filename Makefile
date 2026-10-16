build:
	go vet   ./...
	go test  ./core ./task
	go build ./cmd/goto

install:
	go install -ldflags=-s ./cmd/goto

.PHONY: build install
