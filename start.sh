#!/bin/bash

# Event Trigger Platform - Startup Script
# Starts all services via Docker Compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/deploy/docker-compose.yml"

echo "🚀 Starting Event Trigger Platform..."
echo ""
echo "This will start:"
echo "  - MySQL (localhost:3306)"
echo "  - Kafka + Zookeeper"
echo "  - API Server (localhost:8080)"
echo "  - Scheduler"
echo "  - Workers (2 replicas)"
echo "  - Retention Manager"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    echo "Please start Docker and try again"
    exit 1
fi

# Build and start services
echo "Building Docker images..."
docker compose -f "$COMPOSE_FILE" build

echo ""
echo "Starting services..."
docker compose -f "$COMPOSE_FILE" up -d

echo ""
echo "Waiting for services to be healthy..."
sleep 5

# Show service status
docker compose -f "$COMPOSE_FILE" ps

echo ""
echo "✅ Event Trigger Platform is running!"
echo ""
echo "📍 Access points:"
echo "   API:     http://localhost:8080"
echo "   Health:  http://localhost:8080/health"
echo "   Metrics: http://localhost:8080/metrics"
echo ""
echo "📋 Useful commands:"
echo "   View logs:    docker compose -f deploy/docker-compose.yml logs -f"
echo "   Stop all:     docker compose -f deploy/docker-compose.yml down"
echo "   Restart:      docker compose -f deploy/docker-compose.yml restart"
echo ""
echo "Press Ctrl+C to view logs, or run 'docker compose -f deploy/docker-compose.yml logs -f'"
