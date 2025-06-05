#!/bin/bash

# Health Tracker - Development Environment Stop Script
# Stops PostgreSQL, pgAdmin, and monitoring services
set -e

echo "🛑 Stopping Health Tracker Development Environment..."

# Load environment variables
if [ -f .env ]; then
  export $(cat .env | grep -v '^#' | xargs)
fi

# Build compose command
COMPOSE_CMD="docker-compose -f docker-compose.yaml -f docker-compose.dev.yaml"

echo "🐳 Stopping Docker services..."

# Try graceful shutdown first
$COMPOSE_CMD down

# If that fails, try with --remove-orphans
if [ $? -ne 0 ]; then
  echo "⚠️  Graceful shutdown failed, trying with --remove-orphans..."
  $COMPOSE_CMD down --remove-orphans
fi

# If still failing, try force removal
if [ $? -ne 0 ]; then
  echo "⚠️  Force stopping containers and cleaning up networks..."
  
  # Stop containers individually
  docker stop health-tracker_dev-postgres health-tracker_dev-pgadmin health-tracker_dev-postgres-exporter 2>/dev/null || true
  
  # Remove containers
  docker rm health-tracker_dev-postgres health-tracker_dev-pgadmin health-tracker_dev-postgres-exporter 2>/dev/null || true
  
  # Clean up networks
  docker network rm go-health-tracker_health-tracker-network 2>/dev/null || true
  
  # Prune unused networks
  docker network prune -f
fi

echo "✅ Development environment stopped successfully!"
echo ""
echo "💡 If you encounter network issues, you can also try:"
echo "  - docker system prune -f"
echo "  - docker network prune -f" 