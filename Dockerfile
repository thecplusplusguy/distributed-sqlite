# ABOUTME: Dockerfile for distributed SQLite node container
# ABOUTME: Creates lightweight container for running a single database node in Kubernetes
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o node ./cmd/node

# Final stage - minimal runtime
FROM alpine:latest

# Install ca-certificates for HTTPS requests between nodes
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN adduser -D -s /bin/sh nodeuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/node .

# Change ownership
RUN chown nodeuser:nodeuser /app/node

# Switch to non-root user
USER nodeuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./node"]