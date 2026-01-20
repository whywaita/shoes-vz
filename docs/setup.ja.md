# shoes-vz セットアップガイド

このドキュメントでは、shoes-vz の各コンポーネントのセットアップ手順を説明します。

## 目次

1. [前提条件](#前提条件)
2. [ビルド](#ビルド)
3. [Server のセットアップ](#server-のセットアップ)
4. [Agent のセットアップ](#agent-のセットアップ)
5. [動作確認](#動作確認)
6. [トラブルシューティング](#トラブルシューティング)

## 前提条件

### ハードウェア要件

- **Apple Silicon Mac** (M1, M2, M3 以降)
- **RAM**: 16GB 以上推奨（VM 実行のため）
- **ストレージ**: 100GB 以上の空き容量（VM テンプレート + Runner 用）

### ソフトウェア要件

#### Server 実行環境

- macOS 13.0+ または Linux
- Go 1.21+
- Protocol Buffers コンパイラ（開発時のみ）
  - `brew install buf`

#### Agent 実行環境

- **macOS 26+ (Apple Silicon)**
- **APFS ファイルシステム** (CoW 機能のため)
- **Virtualization.framework の利用権限**
- Go 1.21+

#### Guest VM 要件

- macOS 13+
- SSH サーバー有効化
- GitHub Actions Runner
- shoes-vz-runner-agent

## ビルド

### 1. リポジトリのクローン

```bash
git clone https://github.com/whywaita/shoes-vz.git
cd shoes-vz
```

### 2. 依存関係のインストール

```bash
make deps
```

または手動で：

```bash
go mod download
```

### 3. Protocol Buffers コードの生成

```bash
make proto-generate
```

### 4. ビルド

```bash
make build
```

ビルドされたバイナリは `bin/` ディレクトリに配置されます：

- `bin/shoes-vz-server`
- `bin/shoes-vz-agent`
- `bin/shoes-vz-runner-agent`
- `bin/shoes-vz-client`

### 5. ビルドの確認

```bash
./bin/shoes-vz-server -h
./bin/shoes-vz-agent -h
./bin/shoes-vz-runner-agent -h
./bin/shoes-vz-client -h
```

## Server のセットアップ

### 基本設定

Server は myshoes との連携ポイントとなり、複数の Agent を管理します。

#### 1. 設定ファイルの準備（オプション）

環境変数または起動オプションで設定を指定します。

#### 2. Server の起動

```bash
./bin/shoes-vz-server \
  -grpc-addr :50051 \
  -metrics-addr :9090
```

**オプション:**

- `-grpc-addr`: gRPC サーバーのリッスンアドレス（デフォルト: `:50051`）
- `-metrics-addr`: Prometheus メトリクスのリッスンアドレス（デフォルト: `:9090`）

#### 3. 動作確認

**gRPC の確認:**

```bash
# grpcurl を使用（要インストール: brew install grpcurl）
grpcurl -plaintext localhost:50051 list
```

**メトリクスの確認:**

```bash
curl http://localhost:9090/metrics
```

### systemd での運用（Linux の場合）

`/etc/systemd/system/shoes-vz-server.service`:

```ini
[Unit]
Description=shoes-vz Server
After=network.target

[Service]
Type=simple
User=shoesvz
ExecStart=/opt/shoes-vz/bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

起動:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shoes-vz-server
sudo systemctl start shoes-vz-server
```

### launchd での運用（macOS の場合）

`~/Library/LaunchAgents/com.github.whywaita.shoes-vz-server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/shoes-vz/bin/shoes-vz-server</string>
        <string>-grpc-addr</string>
        <string>:50051</string>
        <string>-metrics-addr</string>
        <string>:9090</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-server.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-server.error.log</string>
</dict>
</plist>
```

起動:

```bash
launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-server.plist
```

## Agent のセットアップ

Agent は macOS ホスト上で動作し、VM の作成・管理を行います。

### 前提条件の確認

#### 1. Virtualization.framework の権限確認

```bash
# 権限の確認
csrutil status
```

`System Integrity Protection status: disabled` または特定の開発者モードが有効である必要があります。

#### 2. APFS ボリュームの確認

```bash
diskutil list
```

テンプレートと Runner を配置するボリュームが APFS であることを確認してください。

### ディレクトリ構造の作成

```bash
# テンプレート用ディレクトリ
sudo mkdir -p /opt/myshoes/vz/templates

# Runner 用ディレクトリ
sudo mkdir -p /opt/myshoes/vz/runners

# 権限設定
sudo chown -R $(whoami):staff /opt/myshoes
```

### SSH キーの準備

Runner VM へのアクセス用 SSH キーを作成します。

```bash
ssh-keygen -t ed25519 -f ~/.ssh/shoes-vz-runner -N ""
```

公開鍵 (`~/.ssh/shoes-vz-runner.pub`) は後でテンプレート作成時に使用します。

### Agent の起動

```bash
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname $(hostname) \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ssh-key ~/.ssh/shoes-vz-runner
```

**オプション:**

- `-server`: Server の gRPC アドレス（デフォルト: `localhost:50051`）
- `-hostname`: Agent のホスト名（デフォルト: システムのホスト名）
- `-max-runners`: 同時実行可能な Runner の最大数（デフォルト: `2`、上限: `2`）
- `-template-path`: VM テンプレートのパス
- `-runners-path`: Runner VM を配置するディレクトリ
- `-ssh-key`: SSH 秘密鍵のパス（オプション）

### launchd での運用

`~/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/shoes-vz/bin/shoes-vz-agent</string>
        <string>-server</string>
        <string>server.example.com:50051</string>
        <string>-hostname</string>
        <string>mac-agent-1</string>
        <string>-max-runners</string>
        <string>2</string>
        <string>-template-path</string>
        <string>/opt/myshoes/vz/templates/macos-26</string>
        <string>-runners-path</string>
        <string>/opt/myshoes/vz/runners</string>
        <string>-ssh-key</string>
        <string>/Users/runner/.ssh/shoes-vz-runner</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-agent.error.log</string>
</dict>
</plist>
```

起動:

```bash
launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist
```

## 動作確認

### 1. Server のログ確認

```bash
# systemd の場合
sudo journalctl -u shoes-vz-server -f

# launchd の場合
tail -f /var/log/shoes-vz-server.log
```

期待されるログ:

```
Starting shoes-vz-server
gRPC address: :50051
Metrics address: :9090
Metrics server listening on :9090
gRPC server listening on :50051
```

### 2. Agent のログ確認

```bash
# launchd の場合
tail -f /var/log/shoes-vz-agent.log
```

期待されるログ:

```
Starting shoes-vz-agent
Server: localhost:50051
Hostname: mac-agent-1
Max runners: 4
Template path: /opt/myshoes/vz/templates/macos-26
Runners path: /opt/myshoes/vz/runners
Connected to server
Agent registered: agent-id-xxx
```

### 3. メトリクスの確認

```bash
curl http://localhost:9090/metrics | grep shoesvz
```

期待される出力例:

```
shoesvz_agents_online 1
shoesvz_capacity_total_runners 4
shoesvz_runners_total{state="creating"} 0
shoesvz_runners_total{state="running"} 0
```

### 4. gRPC 接続の確認

grpcurl を使用して Server への接続を確認:

```bash
# サービス一覧
grpcurl -plaintext localhost:50051 list

# Agent の状態確認（実際の gRPC メソッドに応じて調整）
grpcurl -plaintext localhost:50051 shoes.vz.agent.v1.AgentService/...
```

## トラブルシューティング

### Agent が Server に接続できない

**症状:**
```
Failed to connect to server: connection refused
```

**対処:**

1. Server が起動しているか確認:
   ```bash
   ps aux | grep shoes-vz-server
   ```

2. ポートが開いているか確認:
   ```bash
   lsof -i :50051
   ```

3. ファイアウォール設定を確認:
   ```bash
   # macOS の場合
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

### VM テンプレートが見つからない

**症状:**
```
Template not found: /opt/myshoes/vz/templates/macos-26
```

**対処:**

1. テンプレートディレクトリが存在するか確認:
   ```bash
   ls -la /opt/myshoes/vz/templates/
   ```

2. 必要なファイルが揃っているか確認:
   ```bash
   ls -la /opt/myshoes/vz/templates/macos-26/
   # 必要: Disk.img, AuxiliaryStorage
   ```

3. テンプレートを作成していない場合は [image-build.md](./image-build.md) を参照

### APFS clone が失敗する

**症状:**
```
Failed to clone disk: operation not supported
```

**対処:**

1. APFS ボリュームであることを確認:
   ```bash
   diskutil info /opt/myshoes | grep "Type (Bundle)"
   # APFS であるべき
   ```

2. 同じボリューム上にテンプレートと Runner が配置されているか確認:
   ```bash
   df /opt/myshoes/vz/templates
   df /opt/myshoes/vz/runners
   # 同じマウントポイントであるべき
   ```

### Virtualization.framework のエラー

**症状:**
```
Failed to create VM: not entitled
```

**対処:**

1. コード署名とエンタイトルメントを確認:
   ```bash
   codesign -d --entitlements - ./bin/shoes-vz-agent
   ```

2. 開発者モードが有効か確認:
   ```bash
   DevToolsSecurity -status
   ```

3. 必要に応じて開発者モードを有効化:
   ```bash
   sudo DevToolsSecurity -enable
   ```

### メモリ不足エラー

**症状:**
```
Failed to start VM: insufficient memory
```

**対処:**

1. システムのメモリ使用状況を確認:
   ```bash
   vm_stat
   ```

2. `-max-runners` を減らす:
   ```bash
   # 例: 2 → 1
   -max-runners 1
   ```

3. VM の割り当てメモリを調整（コード修正が必要）:
   ```go
   // internal/agent/vm/vm.go
   // 4GB → 2GB に変更
   2*1024*1024*1024, // 2GB memory
   ```

## 次のステップ

1. **VM テンプレートの作成**: [image-build.md](./image-build.md) を参照して、Golden Template を作成します。
2. **myshoes との連携**: myshoes の設定を行い、shoes-vz を Plugin として登録します。
3. **監視設定**: Prometheus でメトリクスを収集し、Grafana でダッシュボードを作成します。

## 関連ドキュメント

- [image-build.md](./image-build.md) - VM テンプレートの作成手順
- [README.md](../README.md) - プロジェクト概要
- [plans/](../plans/) - 詳細な設計ドキュメント
