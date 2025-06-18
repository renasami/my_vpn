#!/bin/bash

# VPN Server Installation Script
# Usage: ./scripts/install.sh

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

# Check system requirements
check_system() {
    info "システム要件をチェックしています..."
    
    # Check macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        error "このスクリプトはmacOS専用です"
        exit 1
    fi
    
    # Check macOS version
    local macos_version=$(sw_vers -productVersion)
    info "macOS バージョン: $macos_version"
    
    # Check architecture
    local arch=$(uname -m)
    info "アーキテクチャ: $arch"
    
    if [[ "$arch" == "arm64" ]]; then
        log "Apple Silicon (M1/M2) を検出しました"
    elif [[ "$arch" == "x86_64" ]]; then
        log "Intel Mac を検出しました"
    else
        warn "未知のアーキテクチャ: $arch"
    fi
}

# Install Homebrew if not exists
install_homebrew() {
    if ! command -v brew &> /dev/null; then
        info "Homebrewをインストールしています..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        
        # Add to PATH for Apple Silicon
        if [[ "$(uname -m)" == "arm64" ]]; then
            echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
            eval "$(/opt/homebrew/bin/brew shellenv)"
        fi
        
        log "Homebrewのインストールが完了しました"
    else
        log "Homebrewは既にインストールされています"
    fi
}

# Install Go
install_go() {
    if ! command -v go &> /dev/null; then
        info "Goをインストールしています..."
        brew install go
        log "Goのインストールが完了しました"
    else
        local go_version=$(go version | cut -d' ' -f3)
        log "Goは既にインストールされています ($go_version)"
    fi
}

# Install WireGuard Tools
install_wireguard() {
    if ! command -v wg-quick &> /dev/null; then
        info "WireGuard Toolsをインストールしています..."
        brew install wireguard-tools
        log "WireGuard Toolsのインストールが完了しました"
    else
        log "WireGuard Toolsは既にインストールされています"
    fi
}

# Install Node.js (optional)
install_nodejs() {
    if ! command -v node &> /dev/null; then
        info "Node.jsをインストールしています..."
        brew install node
        log "Node.jsのインストールが完了しました"
    else
        local node_version=$(node --version)
        log "Node.jsは既にインストールされています ($node_version)"
    fi
}

# Setup project directories
setup_directories() {
    info "プロジェクトディレクトリを設定しています..."
    
    mkdir -p logs
    mkdir -p config
    mkdir -p tmp
    mkdir -p web/static
    
    # Create default config if not exists
    if [[ ! -f "config/server.conf" ]]; then
        cat > config/server.conf << EOF
# VPN Server Configuration
[server]
port = 8080
interface = wg0
listen_port = 51820
address = 10.0.0.1/24
dns = 1.1.1.1, 8.8.8.8

[database]
path = ./vpn.db

[logging]
level = info
file = ./logs/vpn-server.log
EOF
        log "デフォルト設定ファイルを作成しました: config/server.conf"
    fi
    
    log "ディレクトリ設定が完了しました"
}

# Build the project
build_project() {
    info "プロジェクトをビルドしています..."
    
    # Go dependencies
    go mod tidy
    
    # Build binary
    go build -o vpn-server ./cmd/server/main.go
    
    # Make scripts executable
    chmod +x scripts/*.sh
    
    log "プロジェクトのビルドが完了しました"
}

# Setup system services (optional)
setup_service() {
    local setup_service=""
    echo
    read -p "systemd/launchd サービスとして設定しますか？ (y/N): " setup_service
    
    if [[ "$setup_service" =~ ^[Yy]$ ]]; then
        info "macOS LaunchAgent を設定しています..."
        
        local plist_file="$HOME/Library/LaunchAgents/com.vpnserver.plist"
        
        cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.vpnserver</string>
    <key>ProgramArguments</key>
    <array>
        <string>$PROJECT_ROOT/vpn-server</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$PROJECT_ROOT</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$PROJECT_ROOT/logs/vpn-server.log</string>
    <key>StandardErrorPath</key>
    <string>$PROJECT_ROOT/logs/vpn-server-error.log</string>
</dict>
</plist>
EOF
        
        log "LaunchAgent設定ファイルを作成しました: $plist_file"
        
        info "サービスを有効化するには以下のコマンドを実行してください："
        echo "  launchctl load $plist_file"
        echo "  launchctl start com.vpnserver"
        echo
        info "サービスを無効化するには："
        echo "  launchctl stop com.vpnserver"
        echo "  launchctl unload $plist_file"
    fi
}

# Run tests
run_tests() {
    local run_tests=""
    echo
    read -p "テストを実行しますか？ (y/N): " run_tests
    
    if [[ "$run_tests" =~ ^[Yy]$ ]]; then
        info "テストを実行しています..."
        go test ./... -v || warn "一部のテストが失敗しました（WireGuard関連の可能性があります）"
    fi
}

# Show completion message
show_completion() {
    echo
    echo "🎉 インストールが完了しました！"
    echo "================================="
    echo
    info "起動方法:"
    echo "  sudo ./scripts/start.sh          # 通常起動"
    echo "  sudo ./scripts/start.sh --dev    # 開発モード"
    echo "  sudo ./scripts/start.sh --prod   # プロダクションモード"
    echo
    info "停止方法:"
    echo "  ./scripts/stop.sh                # 通常停止"
    echo "  ./scripts/stop.sh --force        # 強制停止"
    echo
    info "Web UI:"
    echo "  http://localhost:8080"
    echo
    info "設定ファイル:"
    echo "  config/server.conf"
    echo
    info "ログファイル:"
    echo "  logs/vpn-server.log"
    echo
    warn "注意："
    echo "  - VPN機能を使用するにはroot権限が必要です"
    echo "  - 初回起動時にファイアウォール設定の許可が求められる場合があります"
    echo
}

# Show help
show_help() {
    echo "VPN Server インストールスクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/install.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --help     このヘルプを表示"
    echo "  --minimal  最小構成でインストール（Node.js除く）"
    echo
    echo "このスクリプトは以下をインストールします:"
    echo "  - Homebrew (未インストールの場合)"
    echo "  - Go"
    echo "  - WireGuard Tools"
    echo "  - Node.js (フロントエンド開発用)"
    echo
    echo "また、プロジェクトのビルドと初期設定を行います。"
}

# Main logic
main() {
    echo "⚙️  VPN Server Installation Script"
    echo "==================================="
    
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--minimal")
            check_system
            install_homebrew
            install_go
            install_wireguard
            setup_directories
            build_project
            run_tests
            setup_service
            show_completion
            ;;
        *)
            check_system
            install_homebrew
            install_go
            install_wireguard
            install_nodejs
            setup_directories
            build_project
            run_tests
            setup_service
            show_completion
            ;;
    esac
}

# Run main function
main "$@"