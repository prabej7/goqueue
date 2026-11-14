include .envrc
MIGRATION_DIR=./migrations

migrate-create:
	@echo "Creating a new migration..."
	@migrate create -seq -ext sql -dir $(MIGRATION_DIR) $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	@echo "Applying migrations..."
	@migrate -path $(MIGRATION_DIR) -database $(DATABASE_URL) up

migrate-down:
	@echo "Rolling back migrations..."
	@migrate -path $(MIGRATION_DIR) -database $(DATABASE_URL) down

migrate-status:
	@echo "Checking migration status..."
	@migrate -path $(MIGRATION_DIR) -database $(DATABASE_URL) version

migrate-force:
	@echo "Forcing migration version..."
	@migrate -path $(MIGRATION_DIR) -database $(DATABASE_URL) force $(filter-out $@,$(MAKECMDGOALS))
		
test:
	@echo "Running tests..."
	@go test ./...
