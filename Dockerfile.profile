FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Собираем только user сервис
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service ./cmd/user

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app

# Копируем только нужные файлы
COPY --from=builder --chown=app:app /app/user-service .
COPY --from=builder --chown=app:app /app/config ./config
COPY --from=builder --chown=app:app /app/.env ./

USER app
CMD ["./user-service"]