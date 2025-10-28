# Multi-stage build for qb-sync (Go + distroless runtime)
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY ./ /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o qb-sync ./cmd/qb-sync/main.go

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/qb-sync /usr/local/bin/qb-sync
ENTRYPOINT ["/usr/local/bin/qb-sync"]