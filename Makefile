DB_URL=postgres://marouaneboufarouj:@localhost:5432/chirpy?sslmode=disable

.PHONY: reset migrate-up migrate-down

reset: migrate-down migrate-up
	@echo "reset complete"

migrate-up:
	cd sql/schema && goose postgres $(DB_URL) up
	@echo "migration-up done"

migrate-down:
	cd sql/schema && goose postgres $(DB_URL) down
	@echo "migration-down done"
