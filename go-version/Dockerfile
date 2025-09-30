# Multi-stage build for ultra-minimal size
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go modules)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod ./

# Copy go mod files first for better caching
COPY go.mod ./

# Download dependencies
RUN go mod download && go mod tidy

# Copy source code
COPY . .

# Run go mod tidy again to ensure go.sum is complete
RUN go mod tidy

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gcs-proxy .

# Final stage - use distroless for security and minimal size
FROM gcr.io/distroless/static-debian11:latest

# Copy CA certificates for HTTPS requests to GCS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/gcs-proxy /gcs-proxy

# Use non-root user
USER 1001

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
  CMD ["/gcs-proxy", "--health-check"]

# Start the application
ENTRYPOINT ["/gcs-proxy"]