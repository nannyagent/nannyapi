# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/nannyapi ./main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S nannyapi && adduser -S nannyapi -G nannyapi

# Create directories for PocketBase data with proper permissions
RUN mkdir -p /app/pb_data /app/pb_scripts && \
    chown -R nannyapi:nannyapi /app

# Copy binary from builder
COPY --from=builder /app/bin/nannyapi .

# Copy patch scripts (these are part of the application)
COPY --from=builder /app/patch_scripts /app/patch_scripts

# Set ownership
RUN chown -R nannyapi:nannyapi /app

# Switch to non-root user
USER nannyapi

# Environment variables
ENV PB_AUTOMIGRATE=true

# Expose PocketBase default port
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

# Volume mount points for persistent data
# IMPORTANT: Mount these volumes to preserve state across container restarts
# - /app/pb_data: SQLite database and PocketBase data (REQUIRED for persistence)
# - /app/pb_scripts: Custom scripts directory (optional)
VOLUME ["/app/pb_data"]

# Run the application
# --dir: PocketBase data directory (mount this volume!)
# --http: Listen address
CMD ["./nannyapi", "serve", "--dir=/app/pb_data", "--http=0.0.0.0:8090"]
