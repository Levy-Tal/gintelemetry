.PHONY: help start stop restart logs clean traffic test

help: ## Show this help message
	@echo "Full Observability Stack - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

start: ## Start all services (Mimir, Tempo, Loki, Grafana, Collector, App)
	@echo "🚀 Starting observability stack..."
	@docker compose up -d
	@echo ""
	@echo "✅ Services started!"
	@echo ""
	@echo "Access URLs:"
	@echo "  Grafana:     http://localhost:3000"
	@echo "  Application: http://localhost:8080"
	@echo ""
	@echo "Run 'make logs' to view logs"
	@echo "Run 'make traffic' to generate test traffic"

stop: ## Stop all services
	@echo "🛑 Stopping services..."
	@docker compose stop

restart: ## Restart all services
	@echo "🔄 Restarting services..."
	@docker compose restart

logs: ## Show logs from all services
	@docker compose logs -f

logs-app: ## Show logs from the application only
	@docker compose logs -f app

logs-collector: ## Show logs from OTEL collector only
	@docker compose logs -f otel-collector

clean: ## Stop and remove all containers and volumes
	@echo "🧹 Cleaning up..."
	@docker compose down -v
	@echo "✅ Cleanup complete!"

traffic: ## Generate test traffic to the application
	@./generate-traffic.sh

test: ## Run a quick test of all endpoints
	@echo "🧪 Testing endpoints..."
	@echo ""
	@echo "Testing /health..."
	@curl -s http://localhost:8080/health | jq . || echo "❌ Failed"
	@echo ""
	@echo "Testing /hello..."
	@curl -s http://localhost:8080/hello | jq . || echo "❌ Failed"
	@echo ""
	@echo "Testing /process/test123..."
	@curl -s http://localhost:8080/process/test123 | jq . || echo "❌ Failed"
	@echo ""
	@echo "✅ Tests complete!"

status: ## Show status of all services
	@docker compose ps

grafana: ## Open Grafana in browser
	@echo "Opening Grafana..."
	@xdg-open http://localhost:3000 2>/dev/null || open http://localhost:3000 2>/dev/null || echo "Please open http://localhost:3000 in your browser"

build: ## Rebuild the application container
	@echo "🔨 Building application..."
	@docker compose build app

rebuild: build restart ## Rebuild and restart the application
