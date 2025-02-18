build:
	docker compose build

run: build
	docker compose up -d

test: run
	docker exec proxy go test -v
