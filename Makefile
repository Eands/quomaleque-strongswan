.PHONY: up down build logs clean restart

up:
	docker-compose --env-file .env up -d

down:
	docker-compose --env-file .env down

build:
	docker-compose --env-file .env build --no-cache

logs:
	docker-compose --env-file .env logs -f

restart:
	docker-compose --env-file .env restart

clean:
	docker-compose --env-file .env down -v
	rm -rf strongswan_certs strongswan_private webapp_certs
