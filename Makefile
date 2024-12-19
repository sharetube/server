.DEFAULT_GOAL := lint

.PHONY: gen-env-example
gen-env-example:
	sed 's/=.*/=/' .env > .env.example

.PHONY: down
down:
	docker compose down

.PHONY: up
up:
	docker compose up

.PHONY: upd
upd:
	docker compose up -d

.PHONY: build
build:
	docker compose build

.PHONY: run
run:
	docker compose up --build

.PHONY: rund
rund:
	docker compose up --build -d

.PHONY: logs
logs:
	docker logs -f sharetube-server | sed 's/\\n/\n/g'

.PHONY: run-logs
run-logs: rund logs

.PHONY: lint
lint:
	golangci-lint run

.PHONY: format
format:
	gofumpt -l .

.PHONY: test
test:
	go test -v ./internal/app

.PHONY: docker-clean
docker-clean:
	docker compose down --remove-orphans
	docker system prune -af
	docker volume prune -f
	docker network prune -f