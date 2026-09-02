# --- Сборка -----------------------------------------------------------------
FROM golang:1.27-alpine AS builder

WORKDIR /src

# Слой с зависимостями кэшируется отдельно: пока go.mod/go.sum не менялись,
# go mod download при пересборке не повторяется.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO не нужен — pgx это чистый Go. Статический бинарник поедет в alpine.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Запуск -----------------------------------------------------------------
FROM alpine:3.20

RUN adduser -D -u 10001 app
COPY --from=builder /out/server /usr/local/bin/server

USER app
EXPOSE 8080

ENTRYPOINT ["server"]
