#!/bin/bash

# VPN Server Network Setup Script
# Usage: sudo ./scripts/setup-network.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
        error "このスクリプトはroot権限で実行する必要があります"
        error "sudo ./scripts/setup-network.sh を実行してください"
        exit 1
    fi
}

# Get network information
get_network_info() {
    info "ネットワーク情報を取得しています..."
    
    # Get public IP
    PUBLIC_IP=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || echo "取得失敗")
    
    # Get local IP
    LOCAL_IP=$(route get default | grep interface | awk '{print $2}' | xargs ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
    
    # Get network interface
    INTERFACE=$(route get default | grep interface | awk '{print $2}')
    
    echo
    log "ネットワーク設定情報"
    echo "===================="
    info "パブリックIP: $PUBLIC_IP"
    info "ローカルIP: $LOCAL_IP"
    info "メインインターフェース: $INTERFACE"
    echo
}

# Setup firewall rules
setup_firewall() {
    info "ファイアウォール設定を行っています..."
    
    # Enable IP forwarding
    sysctl -w net.inet.ip.forwarding=1
    
    # Make IP forwarding persistent
    if ! grep -q "net.inet.ip.forwarding=1" /etc/sysctl.conf 2>/dev/null; then
        echo "net.inet.ip.forwarding=1" >> /etc/sysctl.conf
        log "IP転送を永続的に有効化しました"
    fi
    
    # Setup pfctl rules for WireGuard
    local pf_conf="/etc/pf.conf"
    local backup_conf="/etc/pf.conf.backup.$(date +%Y%m%d_%H%M%S)"
    
    # Backup current pf.conf
    if [[ -f "$pf_conf" ]]; then
        cp "$pf_conf" "$backup_conf"
        log "既存のpf.confをバックアップしました: $backup_conf"
    fi
    
    # Add WireGuard rules
    local wg_rules="
# WireGuard VPN Rules (Added by VPN Server)
# Allow WireGuard traffic
pass in quick on utun0 all
pass out quick on utun0 all

# Allow WireGuard port
pass in quick proto udp from any to any port 51820

# NAT for VPN clients
nat on $INTERFACE from 10.0.0.0/24 to any -> ($INTERFACE)

# Allow forwarding between interfaces
pass from 10.0.0.0/24 to any keep state
pass from any to 10.0.0.0/24 keep state"

    # Check if WireGuard rules already exist
    if ! grep -q "WireGuard VPN Rules" "$pf_conf" 2>/dev/null; then
        echo "$wg_rules" >> "$pf_conf"
        log "WireGuardルールをpf.confに追加しました"
    else
        warn "WireGuardルールは既に存在します"
    fi
    
    # Reload pfctl rules
    pfctl -f "$pf_conf"
    pfctl -e 2>/dev/null || warn "pfctlは既に有効化されています"
    
    log "ファイアウォール設定が完了しました"
}

# Setup macOS firewall
setup_macos_firewall() {
    info "macOSファイアウォール設定を確認しています..."
    
    # Check if firewall is enabled
    local fw_state=$(/usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate | grep "enabled" || echo "disabled")
    
    if [[ "$fw_state" == *"enabled"* ]]; then
        warn "macOSファイアウォールが有効になっています"
        warn "VPN接続に問題がある場合は、以下の設定を確認してください："
        echo "  1. システム環境設定 > セキュリティとプライバシー > ファイアウォール"
        echo "  2. 'ファイアウォールオプション' をクリック"
        echo "  3. VPNサーバーアプリケーションの接続を許可"
        echo "  または、一時的にファイアウォールを無効化してテスト"
    else
        log "macOSファイアウォールは無効です（VPN接続に有利）"
    fi
}

# Check port accessibility
check_ports() {
    info "ポート接続性をチェックしています..."
    
    # Check if ports are in use
    local vpn_port_in_use=$(lsof -i :51820 2>/dev/null | grep LISTEN || echo "")
    local web_port_in_use=$(lsof -i :8080 2>/dev/null | grep LISTEN || echo "")
    
    if [[ -n "$vpn_port_in_use" ]]; then
        log "WireGuardポート(51820)は既に使用中です"
        echo "$vpn_port_in_use"
    else
        warn "WireGuardポート(51820)はまだ使用されていません"
        info "VPNサーバーを起動してください: sudo ./scripts/start.sh"
    fi
    
    if [[ -n "$web_port_in_use" ]]; then
        log "Web UIポート(8080)は既に使用中です"
        echo "$web_port_in_use"
    else
        warn "Web UIポート(8080)はまだ使用されていません"
        info "VPNサーバーを起動してください: sudo ./scripts/start.sh"
    fi
}

# Test external connectivity
test_connectivity() {
    info "外部接続性をテストしています..."
    
    # Test internet connectivity
    if ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        log "インターネット接続: OK"
    else
        error "インターネット接続: 失敗"
    fi
    
    # Test DNS resolution
    if nslookup google.com >/dev/null 2>&1; then
        log "DNS解決: OK"
    else
        error "DNS解決: 失敗"
    fi
    
    # Check if behind NAT
    if [[ "$PUBLIC_IP" != "$LOCAL_IP" ]]; then
        warn "NAT環境を検出しました"
        warn "ルーターでポート転送設定が必要です:"
        echo "  - 外部ポート: 51820 (UDP)"
        echo "  - 内部ポート: 51820 (UDP)"
        echo "  - 転送先IP: $LOCAL_IP"
    else
        log "パブリックIPアドレスを直接使用しています"
    fi
}

# Generate router configuration guide
generate_router_guide() {
    info "ルーター設定ガイドを生成しています..."
    
    local guide_file="./docs/router-setup-guide.txt"
    mkdir -p docs
    
    cat > "$guide_file" << EOF
ルーター設定ガイド
==================

VPNサーバー情報:
- ローカルIP: $LOCAL_IP
- パブリックIP: $PUBLIC_IP
- WireGuardポート: 51820 (UDP)
- Web UIポート: 8080 (TCP)

ルーター設定手順:
1. ルーターの管理画面にアクセス（通常 192.168.1.1 または 192.168.0.1）
2. 「ポート転送」「NAT設定」「仮想サーバー」などのメニューを探す
3. 以下の設定を追加:

   【WireGuard VPN用】
   - 名前: WireGuard VPN
   - 外部ポート: 51820
   - 内部ポート: 51820
   - プロトコル: UDP
   - 内部IP: $LOCAL_IP
   
   【Web UI用（オプション）】
   - 名前: VPN Web UI
   - 外部ポート: 8080
   - 内部ポート: 8080
   - プロトコル: TCP
   - 内部IP: $LOCAL_IP

4. 設定を保存・適用
5. ルーターを再起動

クライアント接続用情報:
- サーバーアドレス: $PUBLIC_IP:51820
- Web UI: http://$PUBLIC_IP:8080

注意:
- Web UIを外部公開する場合は、強力なパスワードを設定してください
- 可能であれば、Web UIは内部ネットワークからのみアクセスを推奨
EOF

    log "ルーター設定ガイドを生成しました: $guide_file"
}

# Show connection instructions
show_connection_info() {
    echo
    log "VPN接続情報"
    echo "============"
    
    if [[ "$PUBLIC_IP" != "取得失敗" ]]; then
        info "外部からの接続:"
        echo "  サーバーアドレス: $PUBLIC_IP:51820"
        echo "  Web UI: http://$PUBLIC_IP:8080"
        echo
    fi
    
    info "内部ネットワークからの接続:"
    echo "  サーバーアドレス: $LOCAL_IP:51820"
    echo "  Web UI: http://$LOCAL_IP:8080"
    echo
    
    info "次のステップ:"
    echo "  1. VPNサーバーを起動: sudo ./scripts/start.sh"
    echo "  2. Web UIでクライアントを作成"
    echo "  3. QRコードまたは設定ファイルでクライアント接続"
    echo "  4. 外部アクセスが必要な場合はルーター設定を実施"
    echo
    
    warn "セキュリティ推奨事項:"
    echo "  - 強力なパスワードを設定"
    echo "  - 不要なポートは開放しない"
    echo "  - 定期的なバックアップを実施"
    echo "  - ログの監視を実施"
}

# Show help
show_help() {
    echo "VPN Server ネットワーク設定スクリプト"
    echo
    echo "使用方法:"
    echo "  sudo ./scripts/setup-network.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --help       このヘルプを表示"
    echo "  --info-only  情報表示のみ（設定変更なし）"
    echo "  --restore    pfctl設定をバックアップから復元"
    echo
    echo "このスクリプトは以下を実行します:"
    echo "  - ネットワーク情報の表示"
    echo "  - IP転送の有効化"
    echo "  - pfctlファイアウォール設定"
    echo "  - ポート接続性チェック"
    echo "  - ルーター設定ガイド生成"
    echo
    echo "注意: root権限が必要です"
}

# Restore pfctl configuration
restore_pfctl() {
    info "pfctl設定の復元を行います..."
    
    local backup_files=($(ls /etc/pf.conf.backup.* 2>/dev/null | sort -r))
    
    if [[ ${#backup_files[@]} -eq 0 ]]; then
        error "バックアップファイルが見つかりません"
        exit 1
    fi
    
    echo "利用可能なバックアップ:"
    for i in "${!backup_files[@]}"; do
        echo "  $((i+1)). $(basename "${backup_files[$i]}")"
    done
    
    read -p "復元するバックアップ番号を入力 (1-${#backup_files[@]}): " choice
    
    if [[ "$choice" =~ ^[0-9]+$ ]] && [[ "$choice" -ge 1 ]] && [[ "$choice" -le ${#backup_files[@]} ]]; then
        local selected_backup="${backup_files[$((choice-1))]}"
        cp "$selected_backup" /etc/pf.conf
        pfctl -f /etc/pf.conf
        log "pfctl設定を復元しました: $(basename "$selected_backup")"
    else
        error "無効な選択です"
        exit 1
    fi
}

# Main logic
main() {
    echo "🌐 VPN Server Network Setup"
    echo "============================"
    
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--info-only")
            get_network_info
            check_ports
            test_connectivity
            show_connection_info
            exit 0
            ;;
        "--restore")
            check_root
            restore_pfctl
            exit 0
            ;;
        *)
            check_root
            get_network_info
            setup_firewall
            setup_macos_firewall
            check_ports
            test_connectivity
            generate_router_guide
            show_connection_info
            ;;
    esac
}

# Run main function
main "$@"