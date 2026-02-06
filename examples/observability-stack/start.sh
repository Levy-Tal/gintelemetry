#!/bin/bash

set -e

echo "🚀 Starting Full Observability Stack..."
echo ""
echo "This will start:"
echo "  - Grafana Mimir (metrics)"
echo "  - Grafana Tempo (traces)"
echo "  - Grafana Loki (logs)"
echo "  - OpenTelemetry Collector"
echo "  - Grafana (dashboards)"
echo "  - Example Application"
echo ""

# Start all services
docker compose up -d

echo ""
echo "✅ All services started!"
echo ""
echo "🌐 Access URLs:"
echo "  - Grafana:     http://localhost:3000 (no login required)"
echo "  - Application: http://localhost:8080"
echo ""
echo "📊 Try these commands to generate traffic:"
echo "  curl http://localhost:8080/hello"
echo "  curl http://localhost:8080/process/12345"
echo "  curl http://localhost:8080/slow"
echo ""
echo "⏳ Waiting for services to be ready..."
sleep 5

# Wait for Grafana to be ready
echo "Checking Grafana..."
until curl -s http://localhost:3000/api/health > /dev/null 2>&1; do
  echo "  Waiting for Grafana..."
  sleep 2
done

echo "✅ Grafana is ready!"

# Wait for application to be ready
echo "Checking application..."
until curl -s http://localhost:8080/health > /dev/null 2>&1; do
  echo "  Waiting for application..."
  sleep 2
done

echo "✅ Application is ready!"
echo ""
echo "🎉 Everything is up and running!"
echo ""
echo "Next steps:"
echo "  1. Open Grafana: http://localhost:3000"
echo "  2. Click 'Explore' in the left sidebar"
echo "  3. Select 'Tempo' to view traces"
echo "  4. Select 'Loki' to view logs"
echo "  5. Select 'Mimir' to view metrics"
echo ""
echo "To stop: docker compose down"
echo "To clean up: docker compose down -v"
