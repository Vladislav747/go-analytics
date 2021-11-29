.PHONY: build
build:
	go build -v ./cmd/analytics

run:
	go run ./cmd/analytics


run:
	go test -v -race -timeout 30s ./...

.DEFAULT_GOAL := build