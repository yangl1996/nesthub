GOLANGCI_LINT_VERSION ?= v1.51.1

.PHONY: all
all: tidy lint test build

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: format
format: golangci-lint
	$(GOLANGCI_LINT) run --fix ./...

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build ./...

.PHONY: run
run:
	go run ./...

##@ Build Dependencies

.PHONY: download-bin
download-bin: golangci-lint

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || { curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION); }
