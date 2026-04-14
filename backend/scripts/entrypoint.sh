#!/bin/sh

# Generate Swagger documentation
echo "Generating Swagger documentation..."
/go/bin/swag init -g cmd/api/main.go

# Build the application
echo "Building application..."
go build -o main ./cmd/api

# Start the application
echo "Starting application..."
exec ./main
