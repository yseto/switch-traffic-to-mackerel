.PHONY: deps fmt lint build

all: build

deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

build:
	go build
