# ============================================================
# Stage 1: Build the Go binary
# ============================================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /app/server .

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl

# Set timezone to Afghanistan
ENV TZ=Asia/Kabul

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy database schema
COPY --from=builder /app/db_schema.sql ./db_schema.sql

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    chown -R appuser:appgroup /app

USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./server"]
