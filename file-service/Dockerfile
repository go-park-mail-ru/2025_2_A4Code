FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build mkdir -p /app/bin && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/file-service ./file-service/cmd

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder --chown=app:app /app/bin/file-service .
COPY --from=builder --chown=app:app /app/config ./config
COPY --from=builder --chown=app:app /app/.env ./
COPY --from=builder --chown=app:app /app/db ./db
USER app
CMD ["./file-service"]