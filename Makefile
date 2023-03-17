.PHONY: deps fmt lint build

all: build

deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
	go install github.com/tenntenn/testtime/cmd/testtime@latest

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

build:
	go build
test: 
	go test -overlay=`testtime` -v ./...

