gen-env-example:
	sed 's/=.*/=/' .env > .env.example

down:
	docker compose down

up:
	docker compose up

upd:
	docker compose up -d

build:
	docker compose build

run:
	docker compose up --build

rund:
	docker compose up --build -d

logs:
	docker logs -f sharetube-server | sed 's/\\n/\n/g'

run-logs: rund logs

docker-clean:
	docker compose down --remove-orphans
	docker system prune -af
	docker volume prune -f
	docker network prune -f

.PHONY: gen-env-example up upd run rund build logs docker-clean run-logs