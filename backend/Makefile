.PHONY: build run test migrate-up migrate-down migrate-status migrate-create swag gen

# Build the application
build:
	go build -o bin/api cmd/api/main.go
	go build -o bin/migrate cmd/migrate/main.go

# Generate swagger docs
swag:
	swag init -g cmd/api/main.go

# Full codegen: swagger → TypeScript types (run after adding/editing Go handlers)
gen: swag
	cd ../frontend && pnpm --filter @workspace/api-spec run codegen

# Run the application
run:
	go run cmd/api/main.go

# Run tests
test:
	go test -v ./...

# Database migrations
migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down

migrate-status:
	go run cmd/migrate/main.go status

# Create a new migration: make migrate-create name=create_users_table
migrate-create:
	go run cmd/migrate/main.go create $(name) sql
