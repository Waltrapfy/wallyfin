# Phase 1: Go-Code kompilieren
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wallyfin .

# Phase 2: Schlankes Alpine-Linux Image bauen
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/wallyfin .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

EXPOSE 8088

ENTRYPOINT ["/app/wallyfin"]
