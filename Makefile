APP_NAME := jira-pr-mcp
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

.PHONY: build run test lint clean

build:
	go build -ldflags "-X main.version=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)" -o bin/$(APP_NAME) .

run:
	go run .

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
