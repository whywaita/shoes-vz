# shoes-vz

shoes-vz は、macOS 26+（Apple Silicon）上で **Code-Hex/vz（Apple Virtualization Framework の Go バインディング）** を直接利用し、GitHub Actions self-hosted runner を **エフェメラルな macOS VM** として高速に生成・実行・破棄するツール群です。

## 概要

shoes-vz は、APFS の Copy-on-Write（clone）を用いた **CoW 的 Runner 複製** を中核に据え、SSH 到達（SSH possible）を起動完了条件としたときの起動時間を最短化します。

### 主な特徴

- **SSH Ready 最速化**: 起動完了条件を「SSH 接続成功」に統一
- **エフェメラル Runner**: Runner ごとに完全に独立した VM 個体
- **CoW ベースの高速複製**: テンプレートを APFS clone で瞬時に複製
- **macOS 26+ 前提**: Virtualization.framework の最新機能を活用
- **gRPC API 中心設計**: GUI 非依存、myshoes との gRPC 連携

## システム構成

shoes-vz は以下の3つのコンポーネントで構成されます：

### shoes-vz-server (単一インスタンス)
- myshoes との gRPC 連携
- Agent 管理（登録・死活監視）
- Runner スケジューリング
- 全 Runner 状態の集約

### shoes-vz-agent (各 macOS ホストに1つ)
- Virtualization.framework 制御（vz 経由）
- テンプレート管理（clone）
- VM ライフサイクル管理
- Server への状態同期

### shoes-vz-runner-agent (各 Guest macOS VM 内)
- GitHub Actions Runner の状態監視
- HTTP API による状態公開
- ホストへの IP アドレス自動通知
- ホストからの HTTP リクエストによるコマンド実行

## ドキュメント

- **[セットアップガイド](docs/setup.md)** - shoes-vz の各コンポーネントのインストールと設定手順
- **[イメージビルドガイド](docs/image-build.md)** - VM テンプレート（Golden Image）の作成手順
- **[設計ドキュメント](docs/design.md)** - 詳細なアーキテクチャと設計の説明

## ビルド

```bash
# 依存関係のインストール
make deps

# Proto ファイルから Go コードを生成
make proto-generate

# ビルド（entitlement 付きで署名）
make build
```

**注意**: Virtualization Framework を使用するため、ビルド時に自動的に `com.apple.security.virtualization` entitlement を付与して署名します。adhoc signature（`codesign -s -`）を使用しているため、開発者証明書は不要です。

## 実行

### Server

```bash
# gRPCとメトリクスサーバーを起動
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090

# メトリクスを確認
curl http://localhost:9090/metrics
```

### Agent

```bash
# 基本的な起動
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname my-host \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ip-notify-port 8081

# デバッグ用にグラフィックス有効化（GUI ウィンドウが開く）
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname my-host \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ip-notify-port 8081 \
  -enable-graphics
```

### Runner Agent (Guest VM 内)

```bash
# 基本的な起動（.runnerファイルから自動的にrunner IDを取得）
./bin/shoes-vz-runner-agent \
  -listen :8080 \
  -runner-path /tmp/runner \
  -host-ip 192.168.64.1 \
  -agent-port 8081

# runner IDを手動で指定する場合
./bin/shoes-vz-runner-agent \
  -listen :8080 \
  -runner-path /tmp/runner \
  -runner-id my-runner-001 \
  -host-ip 192.168.64.1 \
  -agent-port 8081
```

## テスト

```bash
# 全てのテストを実行
make test

# VM Manager のテスト（テンプレートが必要）
TEST_VM_TEMPLATE=/path/to/template go test -v ./internal/agent/vm/
```

## プロジェクト構造

```
shoes-vz/
├── apis/proto/           # Protocol Buffers 定義
├── cmd/                  # エントリーポイント
│   ├── shoes-vz-server/
│   ├── shoes-vz-agent/
│   ├── shoes-vz-runner-agent/
│   └── shoes-vz-client/
├── internal/             # 内部実装
│   ├── server/          # Server 実装
│   ├── agent/           # Agent 実装
│   ├── monitor/         # Runner Agent 実装
│   └── client/          # myshoes プラグインクライアント実装
├── pkg/model/           # 共有モデル
└── gen/                 # 生成コード
```

## API

### myshoes 連携 API (gRPC)

- `AddInstance`: Runner インスタンスを作成
- `DeleteInstance`: Runner インスタンスを削除

### Agent-Server 間 API (gRPC)

- `RegisterAgent`: Agent の登録
- `Sync`: 双方向ストリームによる状態同期

詳細は `apis/proto/` 配下の `.proto` ファイルを参照してください。

## 実装状況

### 完了済み

- ✅ Protocol Buffers 定義とコード生成
- ✅ shoes-vz-server の基本実装
  - gRPC ハンドラ
  - Agent 管理（登録・状態同期）
  - Runner スケジューリング
  - 状態管理（Store）
- ✅ shoes-vz-agent の基本実装
  - Runner Manager
  - VM Manager（Code-Hex/vz を使用）
  - Server との双方向同期
- ✅ shoes-vz-runner-agent の実装
  - VM 起動時の IP アドレス自動通知（HTTP 経由）
  - .runner ファイル監視による自動 runner ID 取得
  - Runner 状態監視
  - HTTP API
- ✅ VM 操作の実装
  - APFS clone によるテンプレート複製
  - VM 作成・起動・停止・削除
  - SSH 接続チェック
- ✅ ネットワーク実装
  - NAT ネットワーク設定
  - ゲスト VM からホストへの IP アドレス通知（HTTP POST）
  - SSH 接続（IP アドレスベース）
  - HTTP による host-guest 間通信（ポート 8080）
  - IP 通知サーバー（ホスト側、ポート 8081）
- ✅ runner-agent の HTTP API
  - コマンド実行エンドポイント（`/exec`）
  - 状態取得エンドポイント（`/status`）
  - ヘルスチェックエンドポイント（`/health`）
- ✅ Prometheus メトリクス
  - Runner 状態メトリクス（総数、アイドル、ビジー、エラー）
  - Agent 状態メトリクス（オンライン数、キャパシティ）
  - キャパシティメトリクス（使用率、利用可能数）
  - パフォーマンスメトリクス（起動時間、リクエスト処理時間）
  - `/metrics` エンドポイント（ポート 9090）
- ✅ 基本的なテスト

### 次のステップ

1. **VM テンプレートの作成**
   - macOS を VM にインストール
   - SSH 設定とユーザ作成
   - GitHub Actions Runner のインストール
   - shoes-vz-runner-agent の配置と LaunchDaemon 設定

2. **エラーハンドリングの改善**
   - より詳細なエラーメッセージ
   - リカバリ処理
   - タイムアウトの最適化

3. **実際の動作確認**
   - テンプレートを使った統合テスト
   - myshoes との連携テスト
   - HTTP 通信のパフォーマンステスト

## 必要な環境

### ホスト（Agent 実行環境）
- macOS 26+ (Apple Silicon)
- APFS ファイルシステム
- Virtualization.framework の利用権限
- Go 1.21+
- buf CLI（Proto コード生成用）

### ゲスト（VM 内）
- macOS 13+
- GitHub Actions Runner
- shoes-vz-runner-agent（状態監視と HTTP API）

## 主要な依存ライブラリ

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Apple Virtualization Framework の Go バインディング
- [prometheus/client_golang](https://github.com/prometheus/client_golang) - Prometheus メトリクス
- [grpc/grpc-go](https://github.com/grpc/grpc-go) - gRPC 実装

## ライセンス

MIT
