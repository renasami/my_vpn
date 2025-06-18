#!/bin/bash

# VPN Server Backup Script
# Usage: ./scripts/backup.sh [--restore backup_file]

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
BACKUP_DIR="./backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/vpn_backup_$TIMESTAMP.tar.gz"

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

# Create backup
create_backup() {
    info "バックアップを作成しています..."
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR"
    
    # Files and directories to backup
    local backup_items=(
        "vpn.db"
        "config/"
        "logs/"
        "web/static/"
    )
    
    # Check which items exist
    local existing_items=()
    for item in "${backup_items[@]}"; do
        if [[ -e "$item" ]]; then
            existing_items+=("$item")
        fi
    done
    
    if [[ ${#existing_items[@]} -eq 0 ]]; then
        warn "バックアップするファイルが見つかりません"
        exit 1
    fi
    
    # Create tar archive
    tar -czf "$BACKUP_FILE" "${existing_items[@]}"
    
    # Create backup info file
    local info_file="${BACKUP_FILE%.tar.gz}.info"
    cat > "$info_file" << EOF
# VPN Server Backup Information
backup_date=$(date)
hostname=$(hostname)
user=$(whoami)
project_root=$PROJECT_ROOT
backup_items=${existing_items[*]}
backup_size=$(du -h "$BACKUP_FILE" | cut -f1)
EOF
    
    log "バックアップが作成されました: $BACKUP_FILE"
    info "バックアップサイズ: $(du -h "$BACKUP_FILE" | cut -f1)"
    info "バックアップ内容:"
    for item in "${existing_items[@]}"; do
        echo "  - $item"
    done
}

# List backups
list_backups() {
    info "利用可能なバックアップ:"
    echo
    
    if [[ ! -d "$BACKUP_DIR" ]] || [[ -z "$(ls -A "$BACKUP_DIR"/*.tar.gz 2>/dev/null)" ]]; then
        warn "バックアップファイルが見つかりません"
        return
    fi
    
    printf "%-25s %-10s %-20s %s\n" "ファイル名" "サイズ" "作成日時" "説明"
    echo "-------------------------------------------------------------------------"
    
    for backup in "$BACKUP_DIR"/*.tar.gz; do
        if [[ -f "$backup" ]]; then
            local filename=$(basename "$backup")
            local size=$(du -h "$backup" | cut -f1)
            local date=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$backup" 2>/dev/null || stat -c "%y" "$backup" 2>/dev/null | cut -d. -f1)
            local info_file="${backup%.tar.gz}.info"
            local description="VPN設定とデータ"
            
            if [[ -f "$info_file" ]]; then
                local backup_date=$(grep "backup_date=" "$info_file" | cut -d= -f2-)
                if [[ -n "$backup_date" ]]; then
                    description="$backup_date"
                fi
            fi
            
            printf "%-25s %-10s %-20s %s\n" "$filename" "$size" "$date" "$description"
        fi
    done
}

# Restore from backup
restore_backup() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        error "バックアップファイルが見つかりません: $backup_file"
        exit 1
    fi
    
    info "バックアップから復元しています: $backup_file"
    
    # Check if server is running
    if pgrep -f "vpn-server" > /dev/null; then
        warn "サーバーが稼働中です。停止してから復元してください"
        echo "停止コマンド: ./scripts/stop.sh"
        read -p "続行しますか？ (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Create backup of current state
    if [[ -f "vpn.db" ]] || [[ -d "config" ]]; then
        local current_backup="$BACKUP_DIR/pre_restore_$(date +%Y%m%d_%H%M%S).tar.gz"
        warn "現在の状態をバックアップしています: $current_backup"
        
        local current_items=()
        [[ -f "vpn.db" ]] && current_items+=("vpn.db")
        [[ -d "config" ]] && current_items+=("config/")
        [[ -d "logs" ]] && current_items+=("logs/")
        
        if [[ ${#current_items[@]} -gt 0 ]]; then
            mkdir -p "$BACKUP_DIR"
            tar -czf "$current_backup" "${current_items[@]}"
        fi
    fi
    
    # Extract backup
    info "バックアップを展開しています..."
    tar -xzf "$backup_file"
    
    log "復元が完了しました"
    
    # Show restore info
    local info_file="${backup_file%.tar.gz}.info"
    if [[ -f "$info_file" ]]; then
        info "復元したバックアップの情報:"
        while read line; do
            if [[ "$line" =~ ^[^#] ]]; then
                echo "  $line"
            fi
        done < "$info_file"
    fi
    
    warn "復元後はサーバーを再起動してください: ./scripts/start.sh"
}

# Clean old backups
clean_backups() {
    local keep_days="$1"
    if [[ -z "$keep_days" ]]; then
        keep_days=30
    fi
    
    info "$keep_days 日より古いバックアップを削除しています..."
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        info "バックアップディレクトリが存在しません"
        return
    fi
    
    local deleted_count=0
    
    # Find and delete old backup files
    while IFS= read -r -d '' file; do
        rm -f "$file"
        rm -f "${file%.tar.gz}.info"
        ((deleted_count++))
        info "削除: $(basename "$file")"
    done < <(find "$BACKUP_DIR" -name "*.tar.gz" -type f -mtime +$keep_days -print0 2>/dev/null)
    
    if [[ $deleted_count -eq 0 ]]; then
        info "削除対象のバックアップはありませんでした"
    else
        log "$deleted_count 個のバックアップを削除しました"
    fi
}

# Show help
show_help() {
    echo "VPN Server バックアップスクリプト"
    echo
    echo "使用方法:"
    echo "  ./scripts/backup.sh [オプション]"
    echo
    echo "オプション:"
    echo "  --restore FILE    指定したバックアップから復元"
    echo "  --list           利用可能なバックアップを一覧表示"
    echo "  --clean [DAYS]   指定した日数より古いバックアップを削除 (デフォルト: 30日)"
    echo "  --help           このヘルプを表示"
    echo
    echo "例:"
    echo "  ./scripts/backup.sh                              # バックアップ作成"
    echo "  ./scripts/backup.sh --list                       # バックアップ一覧"
    echo "  ./scripts/backup.sh --restore backup_file.tar.gz # 復元"
    echo "  ./scripts/backup.sh --clean 7                    # 7日より古いバックアップを削除"
    echo
    echo "バックアップ対象:"
    echo "  - データベース (vpn.db)"
    echo "  - 設定ファイル (config/)"
    echo "  - ログファイル (logs/)"
    echo "  - 静的ファイル (web/static/)"
}

# Main logic
main() {
    echo "💾 VPN Server Backup Script"
    echo "==========================="
    
    case "$1" in
        "--help"|"-h")
            show_help
            exit 0
            ;;
        "--list"|"-l")
            list_backups
            exit 0
            ;;
        "--restore"|"-r")
            if [[ -z "$2" ]]; then
                error "復元するバックアップファイルを指定してください"
                echo "使用法: $0 --restore backup_file.tar.gz"
                exit 1
            fi
            restore_backup "$2"
            exit 0
            ;;
        "--clean"|"-c")
            clean_backups "$2"
            exit 0
            ;;
        "")
            create_backup
            echo
            list_backups
            ;;
        *)
            error "不明なオプション: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"