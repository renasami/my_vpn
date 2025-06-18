# VPN Server for macOS

M1 MacBook Pro 16GBでWireGuard VPNサーバーを構築・管理するためのGo製アプリケーションです。

## 機能

- **WireGuardサーバー管理**: 起動・停止・設定管理
- **クライアント管理**: VPNクライアントの作成・編集・削除
- **Web UI**: ブラウザからの直感的な操作
- **QRコード生成**: クライアント設定の簡単共有
- **認証システム**: ユーザー登録・ログイン機能
- **モニタリング**: サーバー状態・接続状況の監視

## 必要な環境

- macOS (pfctlサポート)
- Go 1.21+
- WireGuard Tools (`brew install wireguard-tools`)
- Node.js 18+ (Web UI用)

## クイックスタート

### 自動インストール（推奨）
```bash
# 全自動インストール
./scripts/install.sh

# 最小構成インストール（Node.js除く）
./scripts/install.sh --minimal
```

### 手動インストール
```bash
# 1. WireGuard Toolsのインストール
brew install wireguard-tools

# 2. プロジェクトのビルド
go mod tidy
go build -o vpn-server ./cmd/server/main.go

# 3. Web UIの準備（オプション）
cd web/frontend
npm install
npm run build
cd ..

# 4. サーバー起動
sudo ./scripts/start.sh
```

## スクリプトによる管理

### サーバー管理
```bash
# 起動
sudo ./scripts/start.sh          # 通常起動
sudo ./scripts/start.sh --dev    # 開発モード（ホットリロード）
sudo ./scripts/start.sh --prod   # プロダクションモード（バックグラウンド）

# 停止
./scripts/stop.sh                # 通常停止
./scripts/stop.sh --force        # 強制停止

# 状態確認
./scripts/status.sh              # 詳細状態
./scripts/status.sh --simple     # 簡易状態
./scripts/status.sh --logs       # ログのみ表示
```

### Makeコマンド（推奨）
```bash
make help            # 利用可能なコマンド一覧
make install         # 自動インストール
make start           # サーバー起動
make stop            # サーバー停止
make status          # 状態確認
make test            # テスト実行
make backup          # バックアップ作成
```

## 使い方

### Web UIアクセス
```bash
# ローカル
http://localhost:8080

# 外部アクセス（ルーター設定後）
http://[パブリックIP]:8080
```

### 初回セットアップ
1. **ネットワーク設定**: `sudo ./scripts/setup-network.sh`
2. **サーバー起動**: `sudo make start`
3. **ユーザー登録**: Web UIでアカウント作成
4. **クライアント作成**: VPN接続用設定を生成

### 他のPCからのVPN接続

#### 1. 簡単なクライアント作成
```bash
# 対話的にクライアント作成
./scripts/generate-client.sh MyLaptop

# QRコード付きで作成（スマホ用）
./scripts/generate-client.sh MyPhone --qr terminal
./scripts/generate-client.sh MyPhone --qr png
```

#### 2. 各デバイスでの接続
- **Windows**: WireGuardをインストール → 設定ファイルをインポート
- **macOS**: App StoreのWireGuard → 設定ファイルをインポート  
- **iOS/Android**: WireGuardアプリ → QRコードをスキャン
- **Linux**: `sudo wg-quick up client.conf`

#### 3. 外部ネットワークからの接続
```bash
# ルーター設定が必要（自動ガイド生成）
sudo ./scripts/setup-network.sh
```

### クライアント管理
- **Web UI**: ブラウザで簡単管理
- **CLI**: `./scripts/generate-client.sh --list`
- **QRコード**: モバイルデバイス用
- **設定ファイル**: PC・ルーター用

### API使用例
```bash
# サーバー状態確認
curl http://localhost:8080/api/status

# クライアント一覧取得
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/clients

# 新規クライアント作成
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-device"}' \
  http://localhost:8080/api/clients
```

## ディレクトリ構成

```
my_vpn/
├── cmd/main.go              # エントリーポイント
├── internal/
│   ├── auth/               # 認証システム
│   ├── database/           # データベース管理
│   ├── models/             # データモデル
│   ├── monitoring/         # サーバー監視
│   ├── server/             # HTTPサーバー
│   └── wireguard/          # WireGuard制御
├── web/
│   ├── frontend/           # SolidJS SPA
│   └── static/             # ビルド済みフロントエンド
└── tests/                  # テストファイル
```

## 開発者向け

### テスト実行
```bash
# バックエンドテスト
go test ./...

# フロントエンドテスト
cd web/frontend
npm test
```

### 開発サーバー起動
```bash
# バックエンド (ホットリロード)
go run ./cmd/main.go

# フロントエンド開発サーバー
cd web/frontend
npm run dev
```

## 注意事項

- **管理者権限必須**: WireGuardインターフェースとpfctl制御のため
- **ファイアウォール設定**: 自動でpfctlルールを追加/削除
- **ポート使用**: デフォルトで8080(Web)、51820(WireGuard)を使用
- **データベース**: SQLiteファイルとして設定を保存

## トラブルシューティング

### 権限エラー
```bash
sudo ./vpn-server
```

### WireGuardインターフェース削除
```bash
sudo wg-quick down wg0
```

### 設定リセット
```bash
rm -f vpn.db
```