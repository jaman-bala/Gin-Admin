FROM golang:1.27-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

# Копируем скрипт запуска
COPY scripts/entrypoint.sh /app/scripts/entrypoint.sh
RUN chmod +x /app/scripts/entrypoint.sh

# Генерируем документацию при сборке (на всякий случай)
RUN /go/bin/swag init -g cmd/api/main.go

# Собираем приложение
RUN go build -o main ./cmd/api

EXPOSE ${SERVER_PORT}

ENTRYPOINT ["/app/scripts/entrypoint.sh"]
