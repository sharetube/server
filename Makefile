gen-env-example:
	sed 's/=.*/=/' .env > .env.example

up:
	docker compose up

build:
	docker compose build

run:
	docker compose up --build

docker-clean:
	docker compose down --remove-orphans
	docker system prune -af
	docker volume prune -f
	docker network prune -f

.PHONY: gen-env-example up run build docker-clean