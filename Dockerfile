FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/a4mail

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder --chown=app:app /app/main .
COPY --from=builder --chown=app:app /app/config ./config
COPY --from=builder --chown=app:app /app/.env ./
COPY --from=builder --chown=app:app /app/db ./db
USER app
CMD ["./main"]