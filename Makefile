.PHONY: build test lint run migrate-postgres migrate-mssql migrate-oracle \
        docker-build up down logs ps

BINARY=bin/datavault
MAIN=./cmd/datavault
COMPOSE=docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.override.yml --env-file .env

build:
	go build -mod=vendor -o $(BINARY) $(MAIN)

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

run:
	go run $(MAIN)

migrate-postgres:
	psql "$(DATAVAULT_DB_DSN)" -f migrations/postgres/001_init.sql

migrate-mssql:
	sqlcmd -S "$(DATAVAULT_MSSQL_HOST)" -U "$(DATAVAULT_MSSQL_USER)" -P "$(DATAVAULT_MSSQL_PASS)" -d "$(DATAVAULT_MSSQL_DB)" -i migrations/mssql/001_init.sql

migrate-oracle:
	sqlplus "$(DATAVAULT_ORACLE_USER)/$(DATAVAULT_ORACLE_PASS)@$(DATAVAULT_ORACLE_DSN)" @migrations/oracle/001_init.sql

docker-build:
	docker build --target production -t datavault:latest .

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f datavault

ps:
	$(COMPOSE) ps

tidy:
	go mod tidy ; go mod vendor
