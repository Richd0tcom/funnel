# Build stage
FROM golang:1.23-alpine AS builder


RUN apk add --no-cache \
    gcc \
    musl-dev \
    libc-dev \
    librdkafka-dev \
    pkgconf

WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download

# Copy source code selectively (avoid copying unnecessary files)
# COPY *.go ./
# COPY core/ ./core/
COPY . .

# Build with CGO enabled and optimized flags
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s -linkmode external -extldflags "-static"' \
    -a -installsuffix cgo \
    -tags musl \
    -o main .

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies for librdkafka
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    librdkafka \
    wget \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary with proper ownership
COPY --from=builder --chown=appuser:appgroup /app/main .

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Optimized health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run application
CMD ["./main"]