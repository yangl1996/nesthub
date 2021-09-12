LINTERS := -E gci -E gofmt -E whitespace

all: vendor lint build

vendor:
	go mod tidy

lint:
	golangci-lint run $(LINTERS)

format:
	golangci-lint run --fix $(LINTERS)

build:
	go build ./...

.PHONY: all vendor lint format build
