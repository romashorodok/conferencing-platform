env_stage := $(or local)
include  Makefile.$(env_stage)

sqlc:
	~/go/bin/sqlc generate

MIGRATION_NAME=$(or $(MIGRATION), init)
create:
	mkdir -p ./migrations
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) create $(MIGRATION_NAME) sql

up:
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) up
redo:
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) redo
down:
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) down
reset:
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) reset
status:
	~/go/bin/goose -dir ./migrations -table schema_migrations postgres $(DB_URI) status
