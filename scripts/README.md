# Deployment Scripts

このディレクトリには、shoes-vz-agent を macOS ホストに簡単にデプロイするためのスクリプトが含まれています。

## deploy-agent.sh

GitHub Release から shoes-vz-agent バイナリをダウンロードし、LaunchDaemon として登録するスクリプト。

### 必要な権限

このスクリプトは root 権限で実行する必要があります。

```bash
sudo ./scripts/deploy-agent.sh -s SERVER_ADDR [OPTIONS]
```

### 必須パラメータ

- `-s SERVER_ADDR`: Server gRPC アドレス（例: `localhost:50051`）

### オプションパラメータ

- `-v VERSION`: インストールするバージョン（デフォルト: `latest`）
- `-m MAX_RUNNERS`: 最大同時実行ランナー数（デフォルト: `2`）
- `-t TEMPLATE_PATH`: VM テンプレートパス（デフォルト: `/opt/myshoes/vz/templates/macos-26`）
- `-r RUNNERS_PATH`: ランナーディレクトリパス（デフォルト: `/opt/myshoes/vz/runners`）
- `-p IP_NOTIFY_PORT`: IP 通知サーバポート（デフォルト: `8081`）
- `-h`: ヘルプメッセージを表示

### 使用例

#### 最新版をインストール

```bash
sudo ./scripts/deploy-agent.sh -s localhost:50051
```

#### 特定のバージョンをインストール

```bash
sudo ./scripts/deploy-agent.sh -s localhost:50051 -v v1.0.0
```

#### カスタム設定でインストール

```bash
sudo ./scripts/deploy-agent.sh \
  -s prod-server:50051 \
  -v v1.0.0 \
  -m 4 \
  -t /custom/templates \
  -r /custom/runners
```

### インストール先

- バイナリ: `/usr/local/bin/shoes-vz-agent`
- Plist: `/Library/LaunchDaemons/com.github.whywaita.shoes-vz-agent.plist`
- 作業ディレクトリ: `/var/lib/shoes-vz`
- ログ: `/var/log/shoes-vz-agent.log`
- エラーログ: `/var/log/shoes-vz-agent-error.log`

### サービス管理

#### ステータス確認

```bash
sudo launchctl list | grep shoes-vz-agent
```

#### サービス停止

```bash
sudo launchctl unload /Library/LaunchDaemons/com.github.whywaita.shoes-vz-agent.plist
```

#### サービス起動

```bash
sudo launchctl load /Library/LaunchDaemons/com.github.whywaita.shoes-vz-agent.plist
```

#### ログ確認

```bash
# 標準ログ
tail -f /var/log/shoes-vz-agent.log

# エラーログ
tail -f /var/log/shoes-vz-agent-error.log
```

## uninstall-agent.sh

shoes-vz-agent をアンインストールするスクリプト。

### 使用方法

```bash
sudo ./scripts/uninstall-agent.sh [OPTIONS]
```

### オプション

- `--remove-data`: 作業ディレクトリ（`/var/lib/shoes-vz`）も削除
- `--remove-runners`: ランナーディレクトリも削除（確認プロンプトが表示されます）
- `-h, --help`: ヘルプメッセージを表示

### 使用例

#### サービスのみアンインストール

```bash
sudo ./scripts/uninstall-agent.sh
```

#### すべてのデータを削除

```bash
sudo ./scripts/uninstall-agent.sh --remove-data --remove-runners
```

### アンインストールされるもの

- LaunchDaemon サービス
- バイナリ（`/usr/local/bin/shoes-vz-agent`）
- Plist ファイル
- ログファイル
- （オプション）作業ディレクトリ
- （オプション）ランナーディレクトリ

## 事前準備

デプロイ前に以下を確認してください。

1. **VM テンプレートの準備**
   - [Image Build Guide](../docs/image-build.md) を参照してテンプレートを作成
   - デフォルトでは `/opt/myshoes/vz/templates/macos-26` に配置

2. **Server の起動**
   - shoes-vz-server が起動していることを確認
   - gRPC アドレスを確認

3. **必要なディレクトリの作成**
   - デプロイスクリプトが自動で作成しますが、手動で作成することも可能

## トラブルシューティング

### サービスが起動しない

1. エラーログを確認:
   ```bash
   cat /var/log/shoes-vz-agent-error.log
   ```

2. plist の設定を確認:
   ```bash
   cat /Library/LaunchDaemons/com.github.whywaita.shoes-vz-agent.plist
   ```

3. バイナリに実行権限があるか確認:
   ```bash
   ls -la /usr/local/bin/shoes-vz-agent
   ```

4. サーバに接続できるか確認:
   ```bash
   nc -zv SERVER_HOST SERVER_PORT
   ```

### VM が起動しない

1. テンプレートパスが正しいか確認
2. テンプレートディレクトリに必要なファイルがあるか確認
3. Virtualization Framework の権限があるか確認（バイナリに entitlement が付与されているか）

### ログファイルの場所

- 標準出力: `/var/log/shoes-vz-agent.log`
- 標準エラー: `/var/log/shoes-vz-agent-error.log`
