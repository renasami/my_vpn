# VPN Server Makefile

.PHONY: help install build start stop status test clean backup restore

# Default target
help:
	@echo "VPN Server Management Commands"
	@echo "=============================="
	@echo ""
	@echo "Setup Commands:"
	@echo "  make install    - Install dependencies and setup environment"
	@echo "  make build      - Build the VPN server binary"
	@echo ""
	@echo "Server Commands:"
	@echo "  make start      - Start the VPN server"
	@echo "  make start-dev  - Start in development mode"
	@echo "  make start-prod - Start in production mode"
	@echo "  make stop       - Stop the VPN server"
	@echo "  make restart    - Restart the VPN server"
	@echo "  make status     - Show server status"
	@echo ""
	@echo "Development Commands:"
	@echo "  make test       - Run all tests"
	@echo "  make test-go    - Run Go tests only"
	@echo "  make test-fe    - Run frontend tests only"
	@echo "  make clean      - Clean build artifacts"
	@echo ""
	@echo "Backup Commands:"
	@echo "  make backup     - Create backup"
	@echo "  make restore    - Restore from backup (interactive)"
	@echo "  make list-backups - List available backups"
	@echo ""
	@echo "Network & Client Commands:"
	@echo "  make setup-network - Setup network and firewall"
	@echo "  make network-info  - Show network information"
	@echo "  make create-client - Create VPN client config"
	@echo "  make create-client-qr - Create client with QR code"
	@echo "  make list-clients  - List existing clients"
	@echo ""
	@echo "Utility Commands:"
	@echo "  make logs       - Show server logs"
	@echo "  make deps       - Update dependencies"
	@echo "  make lint       - Run linters"
	@echo ""
	@echo "Note: VPN functionality requires sudo privileges"

# Installation and setup
install:
	@echo "🔧 Installing VPN Server..."
	./scripts/install.sh

install-minimal:
	@echo "🔧 Installing VPN Server (minimal)..."
	./scripts/install.sh --minimal

# Build commands
build:
	@echo "🏗️  Building VPN Server..."
	go mod tidy
	go build -o vpn-server ./cmd/server/main.go

build-frontend:
	@echo "🏗️  Building Frontend..."
	cd web/frontend && npm install && npm run build

# Server management
start:
	@echo "🚀 Starting VPN Server..."
	sudo ./scripts/start.sh

start-dev:
	@echo "🚀 Starting VPN Server (Development Mode)..."
	sudo ./scripts/start.sh --dev

start-prod:
	@echo "🚀 Starting VPN Server (Production Mode)..."
	sudo ./scripts/start.sh --prod

stop:
	@echo "🛑 Stopping VPN Server..."
	./scripts/stop.sh

restart: stop start

status:
	@echo "📊 Checking VPN Server Status..."
	./scripts/status.sh

status-simple:
	@echo "📊 Quick Status Check..."
	./scripts/status.sh --simple

# Testing
test:
	@echo "🧪 Running All Tests..."
	go test ./... -v

test-go:
	@echo "🧪 Running Go Tests..."
	go test ./... -v

test-fe:
	@echo "🧪 Running Frontend Tests..."
	cd web/frontend && npm test

test-coverage:
	@echo "🧪 Running Tests with Coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Backup and restore
backup:
	@echo "💾 Creating Backup..."
	./scripts/backup.sh

restore:
	@echo "💾 Restoring from Backup..."
	./scripts/backup.sh --list
	@read -p "Enter backup filename: " backup_file; \
	./scripts/backup.sh --restore "$$backup_file"

list-backups:
	@echo "💾 Available Backups..."
	./scripts/backup.sh --list

clean-backups:
	@echo "💾 Cleaning Old Backups..."
	./scripts/backup.sh --clean

# Network and client management
setup-network:
	@echo "🌐 Setting up network configuration..."
	sudo ./scripts/setup-network.sh

network-info:
	@echo "🌐 Showing network information..."
	./scripts/setup-network.sh --info-only

create-client:
	@echo "📱 Creating VPN client..."
	@read -p "Enter client name: " client_name; \
	./scripts/generate-client.sh "$$client_name"

create-client-qr:
	@echo "📱 Creating VPN client with QR code..."
	@read -p "Enter client name: " client_name; \
	./scripts/generate-client.sh "$$client_name" --qr terminal

list-clients:
	@echo "📱 Listing VPN clients..."
	./scripts/generate-client.sh --list

# Utility commands
logs:
	@echo "📝 Showing Server Logs..."
	./scripts/status.sh --logs

logs-tail:
	@echo "📝 Tailing Server Logs..."
	tail -f logs/vpn-server.log

deps:
	@echo "📦 Updating Dependencies..."
	go mod tidy
	go mod download

deps-fe:
	@echo "📦 Updating Frontend Dependencies..."
	cd web/frontend && npm update

lint:
	@echo "🔍 Running Linters..."
	go fmt ./...
	go vet ./...

# Cleanup
clean:
	@echo "🧹 Cleaning Build Artifacts..."
	rm -f vpn-server
	rm -f coverage.out coverage.html
	rm -rf logs/*.log
	go clean

clean-all: clean
	@echo "🧹 Deep Cleaning..."
	rm -f vpn.db
	rm -rf tmp/*
	cd web/frontend && rm -rf node_modules dist

# Development helpers
dev-setup: install build
	@echo "✅ Development environment ready!"
	@echo "Run 'make start-dev' to start in development mode"

prod-setup: install build build-frontend
	@echo "✅ Production environment ready!"
	@echo "Run 'sudo make start-prod' to start in production mode"

# Docker support (future)
docker-build:
	@echo "🐳 Building Docker Image..."
	@echo "Docker support coming soon..."

docker-run:
	@echo "🐳 Running Docker Container..."
	@echo "Docker support coming soon..."

# Quick commands for common workflows
quick-start: build start

quick-restart: stop build start

quick-test: build test

# Show current configuration
config:
	@echo "⚙️  Current Configuration:"
	@echo "========================="
	@if [ -f config/server.conf ]; then \
		cat config/server.conf; \
	else \
		echo "No configuration file found"; \
	fi

# Environment info
env-info:
	@echo "🔍 Environment Information:"
	@echo "=========================="
	@echo "OS: $$(uname -s)"
	@echo "Architecture: $$(uname -m)"
	@echo "Go Version: $$(go version)"
	@echo "WireGuard: $$(which wg-quick 2>/dev/null || echo 'Not installed')"
	@echo "Node.js: $$(node --version 2>/dev/null || echo 'Not installed')"
	@echo "User: $$(whoami)"
	@echo "Project Root: $$(pwd)"