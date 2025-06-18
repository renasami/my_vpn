# VPNクライアント接続ガイド

他のPC・デバイスからVPNサーバーに接続する方法を説明します。

## 📋 前提条件

1. **VPNサーバーが稼働中**であること
2. **サーバーのパブリックIPアドレス**を確認済み
3. **ファイアウォール設定**が適切であること
4. **クライアントデバイス**にWireGuardがインストール済み

## 🌐 1. サーバーのIPアドレス確認

### パブリックIP確認
```bash
# サーバー側で実行
curl ifconfig.me
# または
curl ipinfo.io/ip
```

### ローカルネットワークIP確認
```bash
# 同一ネットワーク内の場合
ifconfig en0 | grep "inet " | awk '{print $2}'
```

## 🔥 2. ファイアウォール設定

### macOS（サーバー側）
```bash
# システム環境設定 > セキュリティとプライバシー > ファイアウォール
# または以下のコマンドで確認・設定

# 現在の設定確認
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

# WireGuardポート（51820）を開放
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/bin/wg-quick
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblock /usr/local/bin/wg-quick

# Web UI用ポート（8080）を開放
# システム環境設定で手動設定を推奨
```

### ルーターでのポート転送設定
ルーター管理画面で以下を設定：

| 項目 | 値 |
|------|-----|
| 外部ポート | 51820 |
| 内部ポート | 51820 |
| プロトコル | UDP |
| 転送先IP | サーバーのローカルIP |

## 👤 3. VPNクライアント作成

### Web UIでクライアント作成
1. ブラウザで `http://[サーバーIP]:8080` にアクセス
2. ログイン
3. 「Clients」ページで「新規クライアント作成」
4. クライアント名を入力（例：`MyLaptop`）
5. 作成後、QRコードまたは設定ファイルを取得

### CLIでクライアント作成
```bash
# サーバー側で実行
curl -X POST http://localhost:8080/api/clients \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"MyLaptop"}'
```

## 📱 4. クライアントデバイス設定

### Windows
1. **WireGuardをインストール**
   ```
   https://www.wireguard.com/install/
   ```

2. **設定ファイルをインポート**
   - WireGuardアプリを起動
   - 「Add Tunnel」→「Import tunnel(s) from file」
   - `.conf`ファイルを選択

3. **QRコードでインポート**
   - 「Add Tunnel」→「Create from QR code」
   - QRコードをスキャン

### macOS（クライアント）
1. **WireGuardをインストール**
   ```bash
   brew install --cask wireguard-tools
   # または App Store から WireGuard アプリをダウンロード
   ```

2. **設定ファイルでインポート**
   ```bash
   # コマンドラインの場合
   sudo wg-quick up /path/to/client.conf
   
   # アプリの場合
   # WireGuardアプリで「Import Tunnel(s) from File」
   ```

### iOS/Android
1. **WireGuardアプリをインストール**
   - iOS: App Store
   - Android: Google Play Store

2. **QRコードでセットアップ**
   - アプリで「+」→「QRコードをスキャン」
   - Web UIで表示されたQRコードを読み取り

### Linux
1. **WireGuardをインストール**
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install wireguard
   
   # CentOS/RHEL
   sudo yum install epel-release
   sudo yum install wireguard-tools
   ```

2. **設定ファイルを配置**
   ```bash
   sudo cp client.conf /etc/wireguard/wg0.conf
   sudo wg-quick up wg0
   
   # 自動起動設定
   sudo systemctl enable wg-quick@wg0
   ```

## ⚙️ 5. 設定ファイルの例

### クライアント設定ファイル（client.conf）
```ini
[Interface]
PrivateKey = CLIENT_PRIVATE_KEY
Address = 10.0.0.2/32
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = SERVER_PUBLIC_KEY
Endpoint = YOUR_SERVER_IP:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

### 主要設定項目の説明
| 項目 | 説明 |
|------|------|
| `PrivateKey` | クライアントの秘密鍵 |
| `Address` | クライアントのVPN内IPアドレス |
| `DNS` | 使用するDNSサーバー |
| `PublicKey` | サーバーの公開鍵 |
| `Endpoint` | サーバーのIP:ポート |
| `AllowedIPs` | VPN経由でアクセスするIP範囲 |
| `PersistentKeepalive` | 接続維持のためのキープアライブ間隔（秒） |

## 🔧 6. トラブルシューティング

### 接続できない場合

1. **サーバー状態確認**
   ```bash
   # サーバー側で実行
   sudo ./scripts/status.sh
   sudo wg show
   ```

2. **ファイアウォール確認**
   ```bash
   # macOS
   sudo pfctl -sr | grep 51820
   
   # ルーターのポート転送設定も確認
   ```

3. **ネットワーク疎通テスト**
   ```bash
   # クライアント側からサーバーへの疎通確認
   ping YOUR_SERVER_IP
   nc -u YOUR_SERVER_IP 51820
   ```

4. **設定ファイル確認**
   - Endpoint のIPアドレスとポートが正しいか
   - 公開鍵・秘密鍵が正しいか
   - AllowedIPs の設定が適切か

### よくあるエラーと対処法

| エラー | 原因 | 対処法 |
|--------|------|--------|
| `Handshake failed` | 公開鍵が間違っている | 設定ファイルの公開鍵を確認 |
| `Endpoint unreachable` | ネットワーク接続問題 | ファイアウォール・ルーター設定確認 |
| `Permission denied` | 権限不足 | sudoで実行、またはアプリの権限確認 |

### ログの確認方法

**サーバー側:**
```bash
# VPNサーバーログ
./scripts/status.sh --logs

# WireGuardログ（macOS）
sudo dmesg | grep wireguard
sudo log show --predicate 'process == "wg-quick"'
```

**クライアント側:**
```bash
# Windows: WireGuardアプリのログタブ
# macOS: コンソールアプリでWireGuardを検索
# Linux: 
sudo journalctl -u wg-quick@wg0
```

## 🚀 7. 最適化設定

### パフォーマンス向上
```ini
# クライアント設定に追加
[Interface]
MTU = 1420  # MTUサイズ最適化

[Peer]
PersistentKeepalive = 25  # NAT環境では必須
```

### セキュリティ強化
```ini
# 特定のサービスのみVPN経由
[Peer]
AllowedIPs = 192.168.1.0/24  # LAN内のみ
# AllowedIPs = 0.0.0.0/0     # 全てのトラフィック（デフォルト）
```

## 📱 8. 各プラットフォーム別手順まとめ

### クイック接続手順

**iPhone/iPad:**
1. App Store で WireGuard インストール
2. サーバーのWeb UIで QRコード表示
3. WireGuardアプリで QRコードスキャン
4. 接続ボタンをタップ

**Android:**
1. Google Play で WireGuard インストール  
2. QRコードスキャンで設定インポート
3. 接続

**Windows:**
1. https://www.wireguard.com/install/ からダウンロード
2. 設定ファイル（.conf）をインポート
3. Activateで接続

**macOS:**
1. `brew install --cask wireguard-tools`
2. 設定ファイルをダブルクリックでインポート
3. 接続

**Linux:**
1. `sudo apt install wireguard`
2. `sudo wg-quick up /path/to/config.conf`

これで他のPCからVPNサーバーに安全に接続できます！🔐