up: 
	docker compose up -d --force-recreate --build
dev: up
	docker compose run app bash

down:
	docker compose down

restart: down up
