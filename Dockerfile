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

# Version label from build args
ARG VERSION=dev
ARG GRYPE_VERSION=v0.106.0
LABEL org.opencontainers.image.version="${VERSION}"

WORKDIR /app

# Install runtime dependencies (wget required for healthcheck, curl for grype install)
RUN apk add --no-cache ca-certificates tzdata wget curl

# Install grype for vulnerability scanning
RUN curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin ${GRYPE_VERSION}

# Create non-root user for security
RUN addgroup -S nannyapi && adduser -S nannyapi -G nannyapi

# Create directories for PocketBase data and grype cache with proper permissions
RUN mkdir -p /app/pb_data /app/pb_scripts /var/cache/grype/db && \
    chown -R nannyapi:nannyapi /app /var/cache/grype

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
ENV ENABLE_VULN_SCAN=false
ENV GRYPE_DB_CACHE_DIR=/var/cache/grype/db

# Expose PocketBase default port
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/api/health || exit 1

# Volume mount points for persistent data
# IMPORTANT: Mount these volumes to preserve state across container restarts
# - /app/pb_data: SQLite database and PocketBase data (REQUIRED for persistence)
# - /app/pb_scripts: Custom scripts directory (optional)
# - /var/cache/grype: Grype vulnerability database cache (optional, improves scan performance)
VOLUME ["/app/pb_data", "/var/cache/grype"]

# Run the application
# --dir: PocketBase data directory (mount this volume!)
# --http: Listen address
# Use ENABLE_VULN_SCAN=true to enable vulnerability scanning
CMD ./nannyapi serve --dir=/app/pb_data --http=0.0.0.0:8090 $([ "$ENABLE_VULN_SCAN" = "true" ] && echo "--enable-vuln-scan --grype-db-cache-dir=$GRYPE_DB_CACHE_DIR")
