.DEFAULT_GOAL := lint

.PHONY: lint
lint:
	golangci-lint run

.PHONY: format
format:
	gofumpt -l .

.PHONY: test
test:
	go test -v ./...