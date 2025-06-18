#!/bin/bash

# VPN Client Configuration Generator
# Usage: ./scripts/generate-client.sh [client_name] [--qr] [--format=png|terminal|base64]

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
API_BASE="http://localhost:8080/api"
TOKEN_FILE="./.tmp_token"
OUTPUT_DIR="./client-configs"

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

# Check if server is running
check_server() {
    if ! curl -s --connect-timeout 3 "$API_BASE/status" >/dev/null 2>&1; then
        error "VPNサーバーに接続できません"
        error "サーバーが起動しているか確認してください: ./scripts/status.sh"
        exit 1
    fi
}

# Authenticate and get token
authenticate() {
    if [[ -f "$TOKEN_FILE" ]]; then
        local token=$(cat "$TOKEN_FILE")
        # Test if token is still valid
        if curl -s -H "Authorization: Bearer $token" "$API_BASE/clients" >/dev/null 2>&1; then
            echo "$token"
            return
        fi
        rm -f "$TOKEN_FILE"
    fi
    
    echo
    info "認証が必要です"
    read -p "ユーザー名: " username
    read -s -p "パスワード: " password
    echo
    
    local response=$(curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    
    local token=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -z "$token" ]]; then
        error "認証に失敗しました"
        exit 1
    fi
    
    echo "$token" > "$TOKEN_FILE"
    echo "$token"
}

# Create VPN client
create_client() {
    local client_name="$1"
    local token="$2"
    
    info "クライアント '$client_name' を作成しています..."
    
    local response=$(curl -s -X POST "$API_BASE/clients" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"$client_name\"}")
    
    local client_id=$(echo "$response" | grep -o '"id":[0-9]*' | cut -d':' -f2)
    
    if [[ -z "$client_id" ]]; then
        error "クライアントの作成に失敗しました"
        echo "Response: $response"
        exit 1
    fi
    
    log "クライアントを作成しました (ID: $client_id)"
    echo "$client_id"
}

# Get client configuration
get_client_config() {
    local client_id="$1"
    local token="$2"
    
    local config=$(curl -s -H "Authorization: Bearer $token" \
        "$API_BASE/clients/$client_id/config")
    
    if [[ -z "$config" ]]; then
        error "設定ファイルの取得に失敗しました"
        exit 1
    fi
    
    echo "$config"
}

# Get QR code
get_qr_code() {
    local client_id="$1"
    local token="$2"
    local format="$3"
    
    local qr_response=$(curl -s -H "Authorization: Bearer $token" \
        "$API_BASE/clients/$client_id/qr?format=$format")
    
    local qr_data=$(echo "$qr_response" | grep -o '"data":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -z "$qr_data" ]]; then
        error "QRコードの取得に失敗しました"
        exit 1
    fi
    
    echo "$qr_data"
}

# Save configuration file
save_config() {
    local client_name="$1"
    local config="$2"
    
    mkdir -p "$OUTPUT_DIR"
    local config_file="$OUTPUT_DIR/${client_name}.conf"
    
    echo "$config" > "$config_file"
    log "設定ファイルを保存しました: $config_file"
}

# Display QR code
display_qr() {
    local client_id="$1"
    local token="$2"
    local format="$3"
    local client_name="$4"
    
    case "$format" in
        "terminal")
            info "ターミナルQRコード:"
            echo
            get_qr_code "$client_id" "$token" "terminal"
            echo
            ;;
        "png")
            local qr_data=$(get_qr_code "$client_id" "$token" "png")
            local png_file="$OUTPUT_DIR/${client_name}_qr.png"
            
            # Decode base64 to PNG file
            echo "$qr_data" | sed 's/data:image\/png;base64,//' | base64 -d > "$png_file"
            log "QRコード画像を保存しました: $png_file"
            
            # Try to open with default image viewer
            if command -v open >/dev/null; then
                open "$png_file"
            fi
            ;;
        "base64")
            info "Base64 QRコード:"
            get_qr_code "$client_id" "$token" "base64"
            echo
            ;;
        *)
            error "不明なQRコード形式: $format"
            exit 1
            ;;
    esac
}

# Show client instructions
show_instructions() {
    local client_name="$1"
    local config_file="$OUTPUT_DIR/${client_name}.conf"
    
    echo
    log "クライアント接続手順"
    echo "===================="
    
    info "設定ファイル: $config_file"
    echo
    
    info "接続方法:"
    echo "【Windows】"
    echo "  1. https://www.wireguard.com/install/ からWireGuardをダウンロード"
    echo "  2. 'Add Tunnel' → 'Import tunnel(s) from file'"
    echo "  3. $config_file を選択"
    echo "  4. 'Activate' で接続"
    echo
    
    echo "【macOS】"
    echo "  1. App Store から WireGuard をインストール"
    echo "  2. 設定ファイルをダブルクリック、またはアプリでインポート"
    echo "  3. 接続ボタンをクリック"
    echo
    
    echo "【iOS/Android】"
    echo "  1. WireGuardアプリをインストール"
    echo "  2. QRコードをスキャンしてインポート"
    echo "  3. 接続"
    echo
    
    echo "【Linux】"
    echo "  1. sudo apt install wireguard (Ubuntu/Debian)"
    echo "  2. sudo cp $config_file /etc/wireguard/wg0.conf"
    echo "  3. sudo wg-quick up wg0"
    echo
    
    warn "注意事項:"
    echo "  - この設定ファイルは機密情報を含みます"
    echo "  - 安全な方法で送信・保存してください"
    echo "  - 不要になったら削除してください"
}

# List existing clients
list_clients() {
    local token="$1"
    
    info "既存のクライアント一覧:"
    echo
    
    local clients=$(curl -s -H "Authorization: Bearer $token" "$API_BASE/clients")
    
    # Parse JSON response (simple extraction)
    echo "$clients" | grep -o '"id":[0-9]*,"name":"[^"]*"' | while read line; do
        local id=$(echo "$line" | grep -o '"id":[0-9]*' | cut -d':' -f2)
        local name=$(echo "$line" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
        echo "  ID: $id - Name: $name"
    done
}

# Show help
show_help() {
    echo "VPN クライアント設定生成スクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/generate-client.sh [クライアント名] [オプション]"
    echo
    echo "オプション:"
    echo "  --qr [format]    QRコードを表示 (format: terminal, png, base64)"
    echo "  --list          既存クライアント一覧を表示"
    echo "  --help          このヘルプを表示"
    echo
    echo "例:"
    echo "  ./scripts/generate-client.sh MyLaptop"
    echo "  ./scripts/generate-client.sh MyPhone --qr terminal"
    echo "  ./scripts/generate-client.sh Tablet --qr png"
    echo "  ./scripts/generate-client.sh --list"
    echo
    echo "出力:"
    echo "  - 設定ファイル: ./client-configs/[名前].conf"
    echo "  - QRコード: ./client-configs/[名前]_qr.png (PNGの場合)"
    echo
    echo "注意:"
    echo "  - VPNサーバーが起動している必要があります"
    echo "  - 初回実行時は認証情報の入力が必要です"
}

# Main logic
main() {
    echo "📱 VPN Client Configuration Generator"
    echo "====================================="
    
    local client_name=""
    local show_qr=false
    local qr_format="terminal"
    local list_only=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --qr)
                show_qr=true
                if [[ -n "$2" ]] && [[ "$2" != --* ]]; then
                    qr_format="$2"
                    shift
                fi
                shift
                ;;
            --list)
                list_only=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                if [[ -z "$client_name" ]]; then
                    client_name="$1"
                fi
                shift
                ;;
        esac
    done
    
    check_server
    local token=$(authenticate)
    
    if [[ "$list_only" == true ]]; then
        list_clients "$token"
        exit 0
    fi
    
    if [[ -z "$client_name" ]]; then
        error "クライアント名を指定してください"
        echo "使用法: $0 <クライアント名> [オプション]"
        exit 1
    fi
    
    local client_id=$(create_client "$client_name" "$token")
    local config=$(get_client_config "$client_id" "$token")
    
    save_config "$client_name" "$config"
    
    if [[ "$show_qr" == true ]]; then
        display_qr "$client_id" "$token" "$qr_format" "$client_name"
    fi
    
    show_instructions "$client_name"
    
    # Cleanup token file for security
    rm -f "$TOKEN_FILE"
    
    echo
    log "クライアント '$client_name' の設定が完了しました！"
}

# Run main function
main "$@"