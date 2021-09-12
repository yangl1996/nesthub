vendor:
	go mod tidy

lint:
	golangci-lint run

format:
	gofmt -w .
	gci -w .