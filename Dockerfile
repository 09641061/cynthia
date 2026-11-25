# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cynthia ./cmd/api

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/cynthia /app/cynthia
COPY migrations/ /app/migrations/
COPY api/openapi.yaml /app/api/openapi.yaml

RUN chown -R app:app /app

USER app

ENV GIN_MODE=release \
	PORT=3000 \
	MIGRATIONS_DIR=/app/migrations

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
	CMD wget -qO- "http://127.0.0.1:${PORT:-3000}/openapi.yaml" >/dev/null || exit 1

ENTRYPOINT ["/app/cynthia"]
