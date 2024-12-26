.DEFAULT_GOAL := lint

IMAGE_NAME = sharetube/server
CONTAINER_NAME = sharetube-server
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

.PHONY: run
run:
	docker run --name $(CONTAINER_NAME) -p $(PORT):$(PORT) $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: stop
stop:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true

.PHONY: restart
restart: stop build run

.PHONY: clean
clean:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) || true

.PHONY: logs
logs:
	docker logs -f $(CONTAINER_NAME)
