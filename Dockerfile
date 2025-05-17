# Build stage
FROM golang:1.24-alpine AS builder

# Install packages required for CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy and download dependencies (for efficient build cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o /go/bin/health-tracker ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime libraries required for SQLite
RUN apk add --no-cache ca-certificates sqlite tzdata

# Create a non-root user
RUN adduser -D -u 1000 appuser

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /go/bin/health-tracker /app/health-tracker

# Create data directory and set proper ownership
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# Set environment variables
ENV DB_PATH=/app/data/health_tracker.db
ENV PORT=8000
ENV TZ=Asia/Tokyo

# Expose port
EXPOSE 8000

# Switch to non-root user
USER appuser

# Run the application
CMD ["/app/health-tracker"]
