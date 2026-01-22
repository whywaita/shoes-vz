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

- **[セットアップガイド](docs/setup.ja.md)** - shoes-vz の各コンポーネントのインストールと設定手順
- **[イメージビルドガイド](docs/image-build.ja.md)** - VM テンプレート（Golden Image）の作成手順
- **[設計ドキュメント](docs/design.ja.md)** - 詳細なアーキテクチャと設計の説明
- **[AGENTS.md](AGENTS.md)** - 開発者向けガイド（ビルド手順、API リファレンス、トラブルシューティング）

## はじめに

```bash
# 依存関係のインストールとビルド
make deps
make build

# 詳細なビルド手順、実行例、開発ワークフローは AGENTS.md を参照
```

## 必要な環境

### ホスト（Agent 実行環境）
- macOS 26+ (Apple Silicon)
- APFS ファイルシステム
- Virtualization.framework の利用権限

### ゲスト（VM 内）
- macOS 13+
- GitHub Actions Runner
- shoes-vz-runner-agent

## 主要な依存ライブラリ

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Apple Virtualization Framework の Go バインディング
- [prometheus/client_golang](https://github.com/prometheus/client_golang) - Prometheus メトリクス
- [grpc/grpc-go](https://github.com/grpc/grpc-go) - gRPC 実装

## ライセンス

MIT
