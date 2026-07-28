.PHONY: atlas-diff atlas-fmt atlas-status build test

MIGRATION_NAME ?= change

atlas-diff:
	docker compose --profile tools run --rm atlas migrate diff $(MIGRATION_NAME) --env local

atlas-fmt:
	docker compose --profile tools run --rm --no-deps atlas schema fmt schema

atlas-status:
	docker compose run --rm migrate migrate status \
		--dir file:///migrations \
		--url "postgres://alt:alt@postgres:5432/alt?sslmode=disable"

build:
	cd backend && go build ./cmd/alt

test:
	cd backend && go test ./...
