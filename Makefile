.DEFAULT_GOAL := lint

IMAGE_NAME = sharetube-server-dev
IMAGE_TAG = latest
PORT = 8080

.PHONY: lint
lint:
	golangci-lint run

.PHONY: format
format:
	gofumpt -l .

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .