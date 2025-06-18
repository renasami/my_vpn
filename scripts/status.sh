#!/bin/bash

# VPN Server Status Script
# Usage: ./scripts/status.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
BINARY_NAME="vpn-server"
PID_FILE="/tmp/vpn-server.pid"
LOG_FILE="./logs/vpn-server.log"
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

header() {
    echo -e "${CYAN}$1${NC}"
}

# Check server process status
check_process_status() {
    header "📊 プロセス状態"
    echo "=============="
    
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log "サーバーは稼働中です (PID: $pid)"
            
            # Show process details
            local process_info=$(ps -p "$pid" -o pid,ppid,user,%cpu,%mem,vsz,rss,start,time,command --no-headers)
            info "プロセス詳細:"
            echo "  PID    PPID USER    %CPU %MEM    VSZ   RSS  START    TIME COMMAND"
            echo "  $process_info"
            
            # Show listening ports
            local listening_ports=$(lsof -p "$pid" -i -P -n 2>/dev/null | grep LISTEN || echo "なし")
            info "リスニングポート:"
            if [[ "$listening_ports" != "なし" ]]; then
                echo "$listening_ports" | while read line; do
                    echo "  $line"
                done
            else
                echo "  $listening_ports"
            fi
            
        else
            warn "PIDファイルが存在しますが、プロセスが見つかりません"
            rm -f "$PID_FILE"
        fi
    else
        warn "サーバーは停止中です"
    fi
    echo
}

# Check API status
check_api_status() {
    header "🌐 API状態"
    echo "========="
    
    local api_url="http://localhost:$PORT/api/status"
    
    if curl -s -f --connect-timeout 5 "$api_url" > /dev/null 2>&1; then
        log "API は応答しています"
        
        # Get API response
        local api_response=$(curl -s "$api_url" 2>/dev/null)
        info "API レスポンス:"
        echo "$api_response" | head -5
        
    else
        error "API が応答していません"
        warn "URL: $api_url"
    fi
    echo
}

# Check WireGuard interface status
check_wireguard_status() {
    header "🔒 WireGuard状態"
    echo "==============="
    
    if command -v wg &> /dev/null; then
        if [[ $EUID -eq 0 ]]; then
            # Check if wg0 interface exists
            if ip link show wg0 >/dev/null 2>&1; then
                log "WireGuardインターフェース wg0 が存在します"
                
                # Show interface details
                info "インターフェース詳細:"
                wg show wg0 2>/dev/null || warn "インターフェース情報の取得に失敗しました"
                
                # Show interface IP
                local interface_ip=$(ip addr show wg0 2>/dev/null | grep "inet " | awk '{print $2}' || echo "不明")
                info "インターフェースIP: $interface_ip"
                
            else
                warn "WireGuardインターフェース wg0 が見つかりません"
            fi
        else
            warn "WireGuard状態の確認にはroot権限が必要です"
            info "詳細確認: sudo ./scripts/status.sh"
        fi
    else
        warn "WireGuard tools (wg) がインストールされていません"
        info "インストール: brew install wireguard-tools"
    fi
    echo
}

# Check database status
check_database_status() {
    header "💾 データベース状態"
    echo "=================="
    
    local db_file="./vpn.db"
    
    if [[ -f "$db_file" ]]; then
        log "データベースファイルが存在します: $db_file"
        
        local db_size=$(du -h "$db_file" | cut -f1)
        info "ファイルサイズ: $db_size"
        
        local db_modified=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$db_file" 2>/dev/null || stat -c "%y" "$db_file" 2>/dev/null)
        info "最終更新: $db_modified"
        
        # Check if we can access the database
        if command -v sqlite3 &> /dev/null; then
            local table_count=$(sqlite3 "$db_file" "SELECT COUNT(*) FROM sqlite_master WHERE type='table';" 2>/dev/null || echo "不明")
            info "テーブル数: $table_count"
            
            local user_count=$(sqlite3 "$db_file" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "不明")
            info "ユーザー数: $user_count"
            
            local client_count=$(sqlite3 "$db_file" "SELECT COUNT(*) FROM clients;" 2>/dev/null || echo "不明")
            info "クライアント数: $client_count"
        fi
    else
        warn "データベースファイルが見つかりません"
        info "サーバーを起動すると自動的に作成されます"
    fi
    echo
}

# Check log files
check_logs() {
    header "📝 ログ状態"
    echo "=========="
    
    if [[ -f "$LOG_FILE" ]]; then
        log "ログファイルが存在します: $LOG_FILE"
        
        local log_size=$(du -h "$LOG_FILE" | cut -f1)
        info "ファイルサイズ: $log_size"
        
        local log_lines=$(wc -l < "$LOG_FILE")
        info "行数: $log_lines"
        
        info "最新のログエントリ (最後の5行):"
        tail -5 "$LOG_FILE" 2>/dev/null | while read line; do
            echo "  $line"
        done
        
    else
        warn "ログファイルが見つかりません: $LOG_FILE"
    fi
    echo
}

# Check system resources
check_system_resources() {
    header "💻 システムリソース"
    echo "=================="
    
    # Memory usage
    if [[ "$(uname)" == "Darwin" ]]; then
        local memory_info=$(vm_stat | head -4)
        info "メモリ使用状況:"
        echo "$memory_info" | while read line; do
            echo "  $line"
        done
    fi
    
    # Disk usage
    local disk_usage=$(df -h . | tail -1)
    info "ディスク使用状況:"
    echo "  $disk_usage"
    
    # CPU load
    local cpu_load=$(uptime | awk -F'load averages:' '{print $2}')
    info "CPU負荷: $cpu_load"
    
    # Network interfaces
    info "ネットワークインターフェース:"
    ifconfig | grep "inet " | while read line; do
        echo "  $line"
    done
    echo
}

# Check configuration
check_configuration() {
    header "⚙️  設定"
    echo "======="
    
    local config_file="./config/server.conf"
    if [[ -f "$config_file" ]]; then
        log "設定ファイルが存在します: $config_file"
        info "設定内容:"
        cat "$config_file" | while read line; do
            echo "  $line"
        done
    else
        warn "設定ファイルが見つかりません: $config_file"
    fi
    echo
}

# Show help
show_help() {
    echo "VPN Server ステータス確認スクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/status.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --help     このヘルプを表示"
    echo "  --simple   簡易表示"
    echo "  --logs     ログのみ表示"
    echo
    echo "このスクリプトは以下の情報を表示します:"
    echo "  - プロセス状態"
    echo "  - API応答状況"
    echo "  - WireGuard状態"
    echo "  - データベース状態"
    echo "  - ログ状況"
    echo "  - システムリソース"
    echo "  - 設定情報"
}

# Simple status check
simple_status() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo "🟢 サーバー: 稼働中 (PID: $pid)"
        else
            echo "🔴 サーバー: 停止中 (PIDファイルは存在)"
        fi
    else
        echo "🔴 サーバー: 停止中"
    fi
    
    if curl -s -f --connect-timeout 2 "http://localhost:$PORT/api/status" > /dev/null 2>&1; then
        echo "🟢 API: 応答中"
    else
        echo "🔴 API: 応答なし"
    fi
    
    if [[ $EUID -eq 0 ]] && command -v wg &> /dev/null && ip link show wg0 >/dev/null 2>&1; then
        echo "🟢 WireGuard: アクティブ"
    else
        echo "🟡 WireGuard: 状態不明"
    fi
}

# Main logic
main() {
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--simple")
            simple_status
            exit 0
            ;;
        "--logs")
            check_logs
            exit 0
            ;;
        *)
            echo "🔍 VPN Server Status Check"
            echo "=========================="
            echo
            
            check_process_status
            check_api_status
            check_wireguard_status
            check_database_status
            check_logs
            check_system_resources
            check_configuration
            
            echo "✅ ステータスチェック完了"
            ;;
    esac
}

# Run main function
main "$@"