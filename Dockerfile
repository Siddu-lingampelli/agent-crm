# Dockerfile - Multi-stage production build for CRM

# Stage 1: Build frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY web/ ./

# Build production assets
RUN npm run build


# Stage 2: Build Go binary
FROM golang:1.26.0-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build from previous stage
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build binary with version info
ARG VERSION=dev
ARG BUILD_ID=unknown
ARG GIT_SHA=unknown
ARG COMMIT=unknown

ENV CGO_ENABLED=1

RUN go build -ldflags="-X main.version=${VERSION} -X main.buildID=${BUILD_ID} -X main.gitSHA=${GIT_SHA} -s -w" -o /app/crm ./cmd/crm


# Stage 3: Production runtime
FROM alpine:3.20 AS runtime

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    tzdata \
    curl \
    && addgroup -g 1000 -S crm \
    && adduser -u 1000 -S crm -G crm

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/crm /app/crm
COPY --from=frontend-builder /app/web/dist /app/web/dist

# Copy migrations
COPY --from=go-builder /app/db/migrations /app/db/migrations

# Create data directory with correct permissions
RUN mkdir -p /app/data /app/logs && chown -R crm:crm /app

# Switch to non-root user
USER crm

# Environment variables
ENV CRM_DATABASE_PATH=/app/data/crm.db \
    CRM_HTTP_HOST=0.0.0.0 \
    CRM_HTTP_PORT=8080 \
    CRM_LOG_FORMAT=json \
    CRM_LOG_LEVEL=info \
    CRM_ENV=production \
    CRM_SECURITY_SESSION_COOKIE_SECURE=true \
    CRM_SECURITY_CSRF_ENABLED=true \
    GOMEMLIMIT=400MiB

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/crm"]