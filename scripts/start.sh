#!/bin/bash

# VPN Server Start Script
# Usage: ./scripts/start.sh [--dev|--prod|--build]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
BINARY_NAME="vpn-server"
PID_FILE="/tmp/vpn-server.pid"
LOG_FILE="./logs/vpn-server.log"
CONFIG_FILE="./config/server.conf"
PORT=${PORT:-8080}

# Functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        warn "VPN機能を使用するにはroot権限が必要です"
        warn "sudo ./scripts/start.sh を実行してください"
        echo
    fi
}

# Check dependencies
check_dependencies() {
    info "依存関係をチェックしています..."
    
    # Go
    if ! command -v go &> /dev/null; then
        error "Goがインストールされていません"
        exit 1
    fi
    
    # WireGuard tools (optional)
    if ! command -v wg-quick &> /dev/null; then
        warn "WireGuard tools (wg-quick) がインストールされていません"
        warn "VPN機能を使用するには以下のコマンドでインストールしてください:"
        warn "brew install wireguard-tools"
        echo
    fi
    
    # Node.js (for frontend build)
    if [[ "$1" == "--build" ]] && ! command -v npm &> /dev/null; then
        warn "Node.jsがインストールされていません（フロントエンドビルドに必要）"
    fi
}

# Create necessary directories
create_directories() {
    info "必要なディレクトリを作成しています..."
    mkdir -p logs
    mkdir -p config
    mkdir -p tmp
    mkdir -p web/static
}

# Build binary
build_binary() {
    info "バイナリをビルドしています..."
    go mod tidy
    go build -o "$BINARY_NAME" ./cmd/server/main.go
    log "ビルド完了: $BINARY_NAME"
}

# Build frontend (if requested)
build_frontend() {
    if [[ -d "web/frontend" ]] && [[ "$1" == "--build" ]]; then
        info "フロントエンドをビルドしています..."
        cd web/frontend
        if [[ -f "package.json" ]]; then
            npm install
            npm run build 2>/dev/null || warn "フロントエンドのビルドに失敗しました（HTMLUIは利用可能）"
        fi
        cd "$PROJECT_ROOT"
    fi
}

# Check if server is already running
check_running() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            error "サーバーは既に稼働中です (PID: $pid)"
            error "停止するには: ./scripts/stop.sh"
            exit 1
        else
            rm -f "$PID_FILE"
        fi
    fi
}

# Start server
start_server() {
    local mode="$1"
    
    check_running
    
    case "$mode" in
        "--dev")
            info "開発モードでサーバーを起動しています..."
            info "URL: http://localhost:$PORT"
            info "停止するには Ctrl+C を押してください"
            go run ./cmd/server/main.go
            ;;
        "--prod")
            info "プロダクションモードでサーバーを起動しています..."
            nohup ./"$BINARY_NAME" > "$LOG_FILE" 2>&1 &
            echo $! > "$PID_FILE"
            log "サーバーが起動しました (PID: $(cat $PID_FILE))"
            info "URL: http://localhost:$PORT"
            info "ログ: tail -f $LOG_FILE"
            info "停止: ./scripts/stop.sh"
            ;;
        *)
            info "サーバーを起動しています..."
            info "URL: http://localhost:$PORT"
            info "停止するには Ctrl+C を押してください"
            ./"$BINARY_NAME"
            ;;
    esac
}

# Status check
show_status() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log "サーバーは稼働中です (PID: $pid)"
            info "URL: http://localhost:$PORT"
            info "ログ: tail -f $LOG_FILE"
        else
            warn "PIDファイルが存在しますが、プロセスが見つかりません"
            rm -f "$PID_FILE"
        fi
    else
        info "サーバーは停止中です"
    fi
}

# Show help
show_help() {
    echo "VPN Server 起動スクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/start.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --dev      開発モード（ホットリロード）"
    echo "  --prod     プロダクションモード（バックグラウンド実行）"
    echo "  --build    フロントエンドもビルドする"
    echo "  --status   サーバーの状態を確認"
    echo "  --help     このヘルプを表示"
    echo
    echo "例:"
    echo "  sudo ./scripts/start.sh          # 通常起動"
    echo "  sudo ./scripts/start.sh --dev    # 開発モード"
    echo "  sudo ./scripts/start.sh --prod   # プロダクションモード"
    echo "  ./scripts/start.sh --build       # ビルドしてから起動"
    echo "  ./scripts/start.sh --status      # 状態確認"
    echo
    echo "注意:"
    echo "  - VPN機能を使用するにはroot権限が必要です"
    echo "  - WireGuard toolsのインストールを推奨: brew install wireguard-tools"
}

# Main logic
main() {
    echo "🚀 VPN Server Startup Script"
    echo "=============================="
    
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--status")
            show_status
            exit 0
            ;;
        "--dev")
            check_root
            check_dependencies
            create_directories
            start_server "--dev"
            ;;
        "--prod")
            check_root
            check_dependencies "$1"
            create_directories
            build_binary
            build_frontend "$1"
            start_server "--prod"
            ;;
        "--build")
            check_dependencies "$1"
            create_directories
            build_binary
            build_frontend "$1"
            check_root
            start_server
            ;;
        *)
            check_root
            check_dependencies
            create_directories
            if [[ ! -f "$BINARY_NAME" ]]; then
                build_binary
            fi
            start_server
            ;;
    esac
}

# Run main function
main "$@"