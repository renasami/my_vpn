#!/bin/bash

# VPN Server Stop Script
# Usage: ./scripts/stop.sh [--force]

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
PROCESS_NAME="vpn-server"

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

# Stop server gracefully
stop_server() {
    local force="$1"
    
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        
        if ps -p "$pid" > /dev/null 2>&1; then
            info "サーバーを停止しています (PID: $pid)..."
            
            if [[ "$force" == "--force" ]]; then
                kill -9 "$pid"
                log "サーバーを強制終了しました"
            else
                kill "$pid"
                
                # Wait for graceful shutdown
                local count=0
                while ps -p "$pid" > /dev/null 2>&1 && [ $count -lt 10 ]; do
                    sleep 1
                    count=$((count + 1))
                done
                
                if ps -p "$pid" > /dev/null 2>&1; then
                    warn "グレースフル停止に失敗しました。強制終了します..."
                    kill -9 "$pid"
                    log "サーバーを強制終了しました"
                else
                    log "サーバーを正常に停止しました"
                fi
            fi
            
            rm -f "$PID_FILE"
        else
            warn "PIDファイルが存在しますが、プロセスが見つかりません"
            rm -f "$PID_FILE"
        fi
    else
        # Try to find and kill any running vpn-server processes
        local pids=$(pgrep -f "$PROCESS_NAME" 2>/dev/null || true)
        
        if [[ -n "$pids" ]]; then
            warn "PIDファイルは存在しませんが、実行中のプロセスが見つかりました"
            
            for pid in $pids; do
                info "プロセスを停止しています (PID: $pid)..."
                if [[ "$force" == "--force" ]]; then
                    kill -9 "$pid"
                else
                    kill "$pid"
                fi
            done
            
            log "すべてのプロセスを停止しました"
        else
            info "停止するプロセスが見つかりません"
        fi
    fi
}

# Clean up WireGuard interface (if running as root)
cleanup_wireguard() {
    if [[ $EUID -eq 0 ]]; then
        info "WireGuardインターフェースをクリーンアップしています..."
        
        # Check if wg0 interface exists
        if ip link show wg0 >/dev/null 2>&1; then
            wg-quick down wg0 2>/dev/null || warn "WireGuardインターフェースの停止に失敗しました"
            log "WireGuardインターフェースを停止しました"
        fi
    fi
}

# Show help
show_help() {
    echo "VPN Server 停止スクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/stop.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --force    強制終了"
    echo "  --help     このヘルプを表示"
    echo
    echo "例:"
    echo "  ./scripts/stop.sh          # 通常停止"
    echo "  ./scripts/stop.sh --force  # 強制停止"
    echo "  sudo ./scripts/stop.sh     # WireGuardも含めて停止"
    echo
    echo "注意:"
    echo "  - WireGuardインターフェースを停止するにはroot権限が必要です"
}

# Main logic
main() {
    echo "🛑 VPN Server Stop Script"
    echo "========================="
    
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--force")
            stop_server "--force"
            cleanup_wireguard
            ;;
        *)
            stop_server
            cleanup_wireguard
            ;;
    esac
    
    echo
    info "サーバーが停止されました"
}

# Run main function
main "$@"