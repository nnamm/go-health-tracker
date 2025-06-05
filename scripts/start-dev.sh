#!/bin/bash

# Health Tracker - Development Environment Startup Script
set -e

# Parse command line arguments
ENABLE_MONITORING=false

while [[ $# -gt 0 ]]; do
  case $1 in
  --with-monitoring)
    ENABLE_MONITORING=true
    shift
    ;;
  -h | --help)
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  --with-monitoring  Enable PostgreSQL monitoring"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "💡 To run the Go application manually:"
    echo "  go run cmd/server/main.go"
    exit 0
    ;;
  *)
    echo "Unknown option $1"
    exit 1
    ;;
  esac
done

echo "🚀 Starting Health Tracker Development Environment..."

# Check if .env file exists
if [ ! -f .env ]; then
  echo "📝 Creating .env file from .env.example..."
  cp .env.example .env
  echo "✅ .env file created. Please review and modify the settings if needed."
  echo ""
  echo "⚠️  IMPORTANT: Update the following placeholders in .env file:"
  echo "  - DB_PASSWORD: Replace 'YOUR_POSTGRES_PASSWORD_HERE' with actual password"
  echo "  - PGADMIN_PASSWORD: Replace 'YOUR_PGADMIN_PASSWORD_HERE' with actual password"
  echo ""
  echo "Press any key to continue after updating .env file..."
  read -n 1 -s
  echo ""
fi

# Load environment variables
if [ -f .env ]; then
  export $(cat .env | grep -v '^#' | xargs)
fi

# Create necessary directories
echo "📁 Creating necessary directories..."
mkdir -p docker/postgres/logs
mkdir -p docker/pgadmin
mkdir -p sql/migrations
# mkdir -p tmp  # Removed: Only needed for Air live reload tool

# Set proper permissions for PostgreSQL logs
chmod 755 docker/postgres/logs

# Build compose command
COMPOSE_CMD="docker-compose -f docker-compose.yaml -f docker-compose.dev.yaml"

# Add profiles based on options
PROFILES=""
if [ "$ENABLE_MONITORING" = true ]; then
  PROFILES="$PROFILES --profile monitoring"
fi

# Start the services
echo "🐳 Starting Docker services..."
$COMPOSE_CMD $PROFILES up -d

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
# Check if gtimeout is available (from coreutils), otherwise use a simple loop
if command -v gtimeout >/dev/null 2>&1; then
  gtimeout 60 bash -c "until docker exec health-tracker_dev-postgres pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-health_tracker}; do sleep 2; done"
elif command -v timeout >/dev/null 2>&1; then
  timeout 60 bash -c "until docker exec health-tracker_dev-postgres pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-health_tracker}; do sleep 2; done"
else
  # Fallback for macOS without coreutils
  counter=0
  max_attempts=30
  until docker exec health-tracker_dev-postgres pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-health_tracker} 2>/dev/null; do
    sleep 2
    counter=$((counter + 1))
    if [ $counter -ge $max_attempts ]; then
      echo "⚠️  Timeout waiting for PostgreSQL to be ready"
      break
    fi
  done
fi

echo "✅ PostgreSQL is ready!"

# Display service information
echo ""
echo "🎉 Development environment is ready!"
echo ""
echo "📊 Services:"
echo "  - PostgreSQL: localhost:${DB_PORT:-5432}"
echo "  - pgAdmin: http://localhost:${PGADMIN_PORT:-8080}"
echo "    - Email: ${PGADMIN_EMAIL:-admin@example.com}"
echo "    - Password: ${PGADMIN_PASSWORD:-change_this_password}"

if [ "$ENABLE_MONITORING" = true ]; then
  echo "  - PostgreSQL Metrics: http://localhost:9187/metrics"
fi

echo ""
echo "🔧 Useful commands:"
echo "  - View logs: docker-compose logs -f"
echo "  - Stop services: docker-compose down"
echo "  - Force stop with cleanup: docker-compose down --remove-orphans"
echo "  - Connect to DB: docker exec -it health-tracker_dev-postgres psql -U ${DB_USER:-postgres} -d ${DB_NAME:-health_tracker}"

echo ""
echo "🔗 Database Connection String for Go App:"
echo "  postgresql://${DB_USER:-postgres}:${DB_PASSWORD:-postgres}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME:-health_tracker}?sslmode=${DB_SSL_MODE:-disable}"
echo ""
echo "📝 Note: Check the .env file and modify settings as needed for your environment."
echo ""
echo "🚀 Quick start options:"
echo "  - Database only: ./scripts/start-dev.sh"
echo "  - With monitoring: ./scripts/start-dev.sh --with-monitoring"
echo ""
echo "💡 To start your Go application:"
echo "  go run cmd/server/main.go"
