.PHONY: atlas-diff atlas-fmt build test

MIGRATION_NAME ?= change

atlas-diff:
	docker compose --profile tools run --rm atlas migrate diff $(MIGRATION_NAME) \
		--dir file:///db/migrations \
		--to file:///db/schema \
		--dev-url "postgres://alt:alt@schema-dev:5432/alt_schema_dev?sslmode=disable"

atlas-fmt:
	docker compose --profile tools run --rm --no-deps atlas schema fmt /db/schema

build:
	cd backend && go build ./cmd/alt

test:
	cd backend && go test ./...
