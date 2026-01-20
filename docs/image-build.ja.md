# VM テンプレート（Golden Image）構築ガイド

このドキュメントでは、Tart を使用した shoes-vz 用の macOS Tahoe VM テンプレート作成手順を説明します。

## 目次

1. [概要](#概要)
2. [前提条件](#前提条件)
3. [Tart のインストール](#tart-のインストール)
4. [テンプレートの作成](#テンプレートの作成)
5. [テンプレートのテスト](#テンプレートのテスト)
6. [トラブルシューティング](#トラブルシューティング)
7. [カスタマイズ](#カスタマイズ)

## 概要

Golden Template（テンプレートイメージ）は、エフェメラル Runner VM の元となる VM イメージです。このテンプレートから APFS CoW（Copy-on-Write）を使用して高速に Runner VM を複製します。

### Tart を使用する理由

- **高速**: イメージのダウンロードとセットアップが 30〜60 分で完了
- **シンプル**: macOS インストーラ不要、すぐに使える vanilla イメージを利用
- **軽量**: 必要最小限のツールのみをインストール（約 15〜20GB）
- **再現性**: スクリプトで自動化可能

### テンプレートの要件

- macOS 13+
- SSH サーバー有効化
- 専用ユーザーアカウント (`runner`)
- shoes-vz-runner-agent
  - Runner 状態監視機能
  - HTTP API による状態公開
- 基本的な開発ツール（Git, Homebrew）

**重要:** GitHub Actions Runner は **テンプレートに含めません**。Runner のダウンロード、インストール、登録はすべて setup_script で実行されます。

**IP 検出の仕組み:** VM 起動後、host は NAT 範囲 (192.168.64.0/24) をスキャンして SSH ポート (22) への接続性を確認することで、VM の IP アドレスを自動的に発見します。

### テンプレートの構成ファイル

```
template-name.bundle/
├── Disk.img              # VM のディスクイメージ
├── AuxiliaryStorage      # macOS VM の補助ストレージ
└── HardwareModel.json    # ハードウェアモデル情報（Tart から抽出）
```

**HardwareModel.json の形式:**
```json
{
  "hardwareModel": "YnBsaXN0MDDUAQIDBAUGBwpYJHZlcnNpb25..."
}
```

base64 エンコードされたハードウェアモデルデータを含む JSON ファイル。

## 前提条件

### ホスト環境

- macOS 13+ (Apple Silicon)
- 十分なストレージ容量（30GB 以上推奨）
- APFS ボリューム
- Homebrew

### 所要時間

- **ダウンロード**: 10〜15 分（約 10GB）
- **セットアップ**: 15〜30 分
- **合計**: 30〜60 分

## Tart のインストール

```bash
# Homebrew 経由でインストール
brew install cirruslabs/cli/tart

# バージョン確認
tart --version
```

## テンプレートの作成

### 1. Vanilla イメージのクローン

Cirrus Labs が提供する macOS Tahoe vanilla イメージをベースにします。

```bash
# macOS Tahoe をクローン
tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest shoes-vz-template
```

**Vanilla イメージの初期状態:**
- ユーザー: `admin` / パスワード: `admin`
- SSH: 無効
- Homebrew: 未インストール

### 2. VM の起動

```bash
# VM を起動（GUI ウィンドウが開きます）
tart run shoes-vz-template
```

起動後、`admin` / `admin` でログインします。

### 3. SSH の有効化

VM のコンソールで以下を実行:

```bash
# SSH を有効化
sudo systemsetup -setremotelogin on

# 確認
sudo systemsetup -getremotelogin
# 出力: Remote Login: On
```

### 4. VM の IP アドレス確認

**別のターミナル（ホスト側）** で:

```bash
# VM の IP アドレスを取得
IP=$(tart ip shoes-vz-template)
echo "VM IP: $IP"

# SSH 接続テスト
ssh admin@$IP
# パスワード: admin
```

### 5. セットアップファイルの転送

**ホスト側で実行:**

```bash
cd /path/to/shoes-vz

# shoes-vz-runner-agent をビルド
make build

# VM に必要なファイルを転送
IP=$(tart ip shoes-vz-template)

scp scripts/setup-minimal-image.sh admin@$IP:/tmp/
scp bin/shoes-vz-runner-agent admin@$IP:/tmp/

# SSH 公開鍵を転送（オプション）
scp ~/.ssh/id_ed25519.pub admin@$IP:/tmp/ssh_public_key
```

### 6. セットアップスクリプトの実行

**VM に SSH 接続:**

```bash
ssh admin@$IP
```

**VM 内でスクリプトを実行:**

```bash
# 実行権限を付与
chmod +x /tmp/setup-minimal-image.sh

# スクリプトを実行
/tmp/setup-minimal-image.sh
```

**スクリプトが実行する内容:**
- SSH の有効化確認
- runner ユーザーの作成（UID: 502）
- SSH 公開鍵の配置（`/tmp/ssh_public_key` が存在する場合）
- Homebrew のインストール
- 基本ツールのインストール（git, curl, wget, jq, yq）
- shoes-vz-runner-agent の配置（`/usr/local/bin/`）
- LaunchAgent の設定（自動起動、IP 通知機能付き）
- システムクリーンアップ（キャッシュ、ログ削除）
- Spotlight 無効化（起動高速化）

完了後、以下のメッセージが表示されます:

```
=== Setup complete ===
You can now shutdown the VM with: sudo shutdown -h now
```

### 7. VM のシャットダウン

**VM 内で実行:**

```bash
sudo shutdown -h now
```

### 8. テンプレート形式に変換

**ホスト側で実行:**

```bash
# テンプレートディレクトリを作成
sudo mkdir -p /opt/myshoes/vz/templates/macos-tahoe

# Tart の VM イメージをテンプレート形式にコピー
sudo cp ~/.tart/vms/shoes-vz-template/disk.img /opt/myshoes/vz/templates/macos-tahoe/Disk.img
sudo cp ~/.tart/vms/shoes-vz-template/nvram.bin /opt/myshoes/vz/templates/macos-tahoe/AuxiliaryStorage

# HardwareModel.json を作成（必須）
if [ -f ~/.tart/vms/shoes-vz-template/config.json ]; then
    # Tart の config.json から hardwareModel を抽出して JSON 形式で保存
    jq '{hardwareModel: .hardwareModel}' ~/.tart/vms/shoes-vz-template/config.json | \
        sudo tee /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null
else
    echo "Error: Tart config.json not found. Cannot create HardwareModel.json"
    exit 1
fi

# 正しい形式で作成されたか確認
if ! jq -e '.hardwareModel' /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null 2>&1; then
    echo "Error: HardwareModel.json is not in the correct format"
    exit 1
fi

# 権限設定
sudo chown -R $(whoami):staff /opt/myshoes/vz/templates/macos-tahoe
chmod 644 /opt/myshoes/vz/templates/macos-tahoe/*

# ディスクサイズを確認
ls -lh /opt/myshoes/vz/templates/macos-tahoe/

# ファイル構成を確認（期待される出力）
# Disk.img            # 20GB 程度
# AuxiliaryStorage    # 数 MB
# HardwareModel.json  # 1KB 程度

# HardwareModel.json の内容を確認
cat /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
# 期待される出力: {"hardwareModel":"YnBsaXN0MDDUAQIDBAUGBwpY..."}
```

**重要な確認項目:**
- ✅ `Disk.img` が存在し、サイズが 20GB 程度
- ✅ `AuxiliaryStorage` が存在
- ✅ `HardwareModel.json` が JSON 形式で `hardwareModel` キーを含む

### 9. テンプレートメタデータの作成（オプション）

```bash
cat > /opt/myshoes/vz/templates/macos-tahoe/TemplateMetadata.json << 'EOF'
{
  "name": "macos-tahoe",
  "version": "15.x",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "description": "macOS Tahoe vanilla template for GitHub Actions self-hosted runner",
  "base_image": "ghcr.io/cirruslabs/macos-tahoe-vanilla:latest",
  "cpu_count": 2,
  "memory_gb": 4,
  "disk_size_gb": 20,
  "features": [
    "SSH enabled",
    "runner user created",
    "shoes-vz-runner-agent installed",
    "HTTP API for monitoring",
    "Homebrew installed",
    "Basic tools (git, curl, wget, jq, yq)"
  ],
  "note": "GitHub Actions Runner will be installed via setup_script at runtime"
}
EOF
```

## テンプレートのテスト

### 1. shoes-vz-agent でテスト

**別のターミナルで shoes-vz-server を起動:**

```bash
cd /path/to/shoes-vz
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090
```

**shoes-vz-agent を起動:**

```bash
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname test-agent \
  -max-runners 1 \
  -template-path /opt/myshoes/vz/templates/macos-tahoe \
  -runners-path /tmp/test-runners \
  -ssh-key ~/.ssh/id_ed25519
```

**期待されるログ:**

```
Starting shoes-vz-agent
Server: localhost:50051
Hostname: test-agent
Max runners: 1
Template path: /opt/myshoes/vz/templates/macos-tahoe
Runners path: /tmp/test-runners
Agent registered with ID: xxx
```

### 2. VM 作成テスト

Go のテストを実行:

```bash
cd /path/to/shoes-vz

# テンプレートパスを指定してテスト
TEST_VM_TEMPLATE=/opt/myshoes/vz/templates/macos-tahoe \
  go test -v ./internal/agent/vm/ -run TestVMManager_Create
```

**期待される出力:**

```
=== RUN   TestVMManager_Create
Starting VM for runner xxx...
Waiting for VM to reach running state...
VM state: VirtualMachineStateRunning
VM is now running, discovering guest IP via TCP/IP...
IP discovery attempt 1...
Trying common NAT IPs...
Found guest IP: 192.168.64.2
Guest IP discovered: 192.168.64.2
--- PASS: TestVMManager_Create (30.00s)
PASS
```

### 3. SSH 接続確認

VM が起動した後、SSH 接続をテスト:

```bash
# runner-agent のログから IP アドレスを確認
ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2 whoami
# 出力: runner
```

### 4. runner-agent の動作確認

```bash
# runner-agent のログを確認
ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2 tail -f /Users/runner/runner-agent.log
```

**期待されるログ:**

```
Starting shoes-vz-runner-agent
Starting on TCP, listen_addr=:8080
Using runner path, path=/Users/runner/actions-runner
Starting HTTP server on :8080
```

## トラブルシューティング

### Tart のダウンロードが遅い

**症状:**
イメージのダウンロードに 30 分以上かかる

**対処:**

```bash
# ダウンロードの進行状況を確認
tart list

# キャッシュをクリア
rm -rf ~/.tart/cache/

# 別のミラーを試す（該当する場合）
```

### SSH 接続ができない

**症状:**
```
ssh: connect to host [IP] port 22: Connection refused
```

**対処:**

1. SSH が有効になっているか確認（VM 内）:
   ```bash
   sudo systemsetup -getremotelogin
   ```

2. ファイアウォールを確認（VM 内）:
   ```bash
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

3. SSH サービスを再起動（VM 内）:
   ```bash
   sudo launchctl stop com.openssh.sshd
   sudo launchctl start com.openssh.sshd
   ```

### runner-agent が起動しない

**症状:**
```
Failed to start runner-agent
```

**対処:**

1. バイナリが存在するか確認（VM 内）:
   ```bash
   ls -la /usr/local/bin/shoes-vz-runner-agent
   ```

2. LaunchAgent のログを確認（VM 内）:
   ```bash
   tail -f /Users/runner/runner-agent.error.log
   ```

3. 手動起動でテスト（VM 内）:
   ```bash
   sudo -u runner /usr/local/bin/shoes-vz-runner-agent \
     -listen :8080 \
     -runner-path /Users/runner/actions-runner
   ```

4. LaunchAgent を再読み込み（VM 内）:
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-runner-agent.plist
   launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-runner-agent.plist
   ```

### IP アドレスが検出されない

**症状:**
```
VM is now running, discovering guest IP via TCP/IP...
IP discovery attempt 1...
Trying common NAT IPs...
timeout discovering guest IP after 3 minutes
```

**対処:**

1. SSH サーバーが起動しているか確認（VM 内）:
   ```bash
   sudo launchctl list | grep sshd
   ```

2. ネットワークインターフェースを確認（VM 内）:
   ```bash
   ifconfig | grep "inet "
   ```

3. ファイアウォールが SSH を許可しているか確認（VM 内）:
   ```bash
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

4. ホストから手動で SSH 接続を試す:
   ```bash
   ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2
   ```

### HardwareModel.json が見つからない

**症状:**
```
hardware model not found in template: /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
```

**原因:**
HardwareModel.json が存在しないか、正しい形式で作成されていない。

**対処:**

1. ファイルが存在するか確認:
   ```bash
   ls -la /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

2. ファイル内容を確認:
   ```bash
   cat /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

   期待される形式:
   ```json
   {
     "hardwareModel": "YnBsaXN0MDDUAQIDBAUGBwpY..."
   }
   ```

3. 存在しない、または形式が間違っている場合は再作成:
   ```bash
   # Tart VM から抽出
   jq '{hardwareModel: .hardwareModel}' ~/.tart/vms/shoes-vz-template/config.json | \
       sudo tee /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null

   # 正しく作成されたか確認
   jq -e '.hardwareModel' /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

4. Tart VM が存在しない場合:
   ```bash
   # Tart VM を再作成
   tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest shoes-vz-template

   # 上記手順を実行
   ```

### テンプレートサイズが大きい

**症状:**
Disk.img が 30GB 以上

**対処:**

1. 不要なファイルを削除（VM 内、シャットダウン前に実行）:
   ```bash
   # Homebrew のキャッシュ削除
   brew cleanup -s

   # Xcode キャッシュ削除（インストールしている場合）
   rm -rf ~/Library/Developer/Xcode/DerivedData/*

   # システムログ削除
   sudo rm -rf /var/log/*
   sudo rm -rf ~/Library/Logs/*
   ```

2. ディスクを圧縮（VM 内）:
   ```bash
   # ゼロ埋め
   sudo dd if=/dev/zero of=/tmp/zero.dat bs=1m || true
   sudo rm /tmp/zero.dat
   ```

3. Tart で最適化（ホスト側）:
   ```bash
   # VM を停止後
   tart prune shoes-vz-template
   ```

## カスタマイズ

### Xcode Command Line Tools のインストール

Xcode が必要な場合、セットアップスクリプトに以下を追加:

```bash
echo "=== Installing Xcode Command Line Tools ==="
xcode-select --install

# インストール完了を待つ
until xcode-select -p &> /dev/null; do
  sleep 5
done
```

### 追加ツールのインストール

Homebrew でツールを追加:

```bash
echo "=== Installing additional tools ==="
brew install \
  node \
  python@3.11 \
  go \
  rust \
  docker
```

### カスタムユーザーの追加

runner 以外のユーザーを追加:

```bash
echo "=== Creating custom user ==="
sudo dscl . -create /Users/myuser
sudo dscl . -create /Users/myuser UserShell /bin/bash
sudo dscl . -create /Users/myuser RealName "My User"
sudo dscl . -create /Users/myuser UniqueID 503
sudo dscl . -create /Users/myuser PrimaryGroupID 20
sudo dscl . -create /Users/myuser NFSHomeDirectory /Users/myuser
sudo mkdir -p /Users/myuser
sudo chown myuser:staff /Users/myuser
```

### セットアップスクリプトのカスタマイズ

`scripts/setup-minimal-image.sh` をコピーして独自のスクリプトを作成:

```bash
cp scripts/setup-minimal-image.sh scripts/setup-custom.sh

# スクリプトを編集
vim scripts/setup-custom.sh

# VM に転送して実行
scp scripts/setup-custom.sh admin@$IP:/tmp/
ssh admin@$IP 'chmod +x /tmp/setup-custom.sh && /tmp/setup-custom.sh'
```

## ベストプラクティス

### 1. テンプレートのバージョン管理

```bash
# テンプレート名にバージョンを含める
/opt/myshoes/vz/templates/
├── macos-tahoe-v1/
├── macos-tahoe-v2/
└── macos-tahoe-latest -> macos-tahoe-v2  # シンボリックリンク

# バージョン管理
ln -sf macos-tahoe-v2 /opt/myshoes/vz/templates/macos-tahoe-latest
```

### 2. 定期的なテンプレート更新

```bash
#!/bin/bash
# update-template.sh - テンプレート更新スクリプト

# 現在の日付をバージョンとして使用
VERSION=$(date +%Y%m%d)
TEMPLATE_NAME="macos-tahoe-$VERSION"

# 新しい vanilla イメージをクローン
tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest $TEMPLATE_NAME

# セットアップ処理...
# （上記の手順を自動化）

# 古いテンプレートを削除（3世代より古いもの）
# ...
```

### 3. セキュリティ設定

```bash
# ファイアウォールを有効化（VM 内）
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on

# Gatekeeper を有効化（VM 内）
sudo spctl --master-enable

# FileVault を無効化（パフォーマンスのため、推奨）
# システム設定 > プライバシーとセキュリティ > FileVault
```

### 4. テストの自動化

```bash
#!/bin/bash
# test-template.sh

TEMPLATE_PATH="/opt/myshoes/vz/templates/macos-sequoia"

# テンプレートの存在確認
if [ ! -d "$TEMPLATE_PATH" ]; then
  echo "❌ Template not found: $TEMPLATE_PATH"
  exit 1
fi

# 必須ファイルの確認
for file in Disk.img AuxiliaryStorage; do
  if [ ! -f "$TEMPLATE_PATH/$file" ]; then
    echo "❌ Missing file: $file"
    exit 1
  fi
done

# VM 作成テスト
echo "🧪 Testing VM creation..."
TEST_VM_TEMPLATE="$TEMPLATE_PATH" \
  go test -v ./internal/agent/vm/ -run TestVMManager_Create

if [ $? -eq 0 ]; then
  echo "✅ Template test passed"
else
  echo "❌ Template test failed"
  exit 1
fi
```

## 次のステップ

テンプレートが正常に動作することを確認したら:

1. **Agent の本番起動**: [setup.md](./setup.md) を参照して Agent を本番環境で起動
2. **myshoes との連携**: shoes-vz-client を使って myshoes と連携
3. **監視とメトリクス**: Prometheus でメトリクスを収集し、Grafana でダッシュボードを作成

## 関連ドキュメント

- [setup.md](./setup.md) - shoes-vz のセットアップ手順
- [README.md](../README.md) - プロジェクト概要
- [Tart 公式ドキュメント](https://tart.run/)
- [Cirrus Labs VM Images](https://github.com/cirruslabs/macos-image-templates)
- [Apple Virtualization Framework Documentation](https://developer.apple.com/documentation/virtualization)
