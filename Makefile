# Kenbun Gateway - Observation Haki for AI Infrastructure

.PHONY: help up down gateway ui simulate clean

help:
	@echo "👁️ Kenbun Gateway - Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make up         Start all infrastructure (Postgres, Redis Cluster, Kafka, ClickHouse)"
	@echo "  make down       Stop all infrastructure"
	@echo "  make gateway    Run the Go Gateway (Data Plane)"
	@echo "  make ui         Run the Next.js Dashboard (Control Plane)"
	@echo "  make simulate   Run the Haki Simulator (Generate load)"
	@echo "  make clean      Remove build artifacts and Docker volumes"

up:
	docker compose up -d

down:
	docker compose down

gateway:
	go run cmd/gateway/main.go

ui:
	cd ui && bun run dev

simulate:
	go run scripts/simulate.go

clean:
	rm -f gateway
	docker compose down -v
