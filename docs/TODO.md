# TODO

shoes-vz の今後実装予定の機能と改善項目をまとめたドキュメントです。

## 機能追加

### リソースタイプ管理

**概要**:
shoes-vz-server でリソースタイプ（small, medium, large 等）を VM リソース構成にマッピングする機能。

**詳細**:
- myshoes からの `AddInstanceRequest` に含まれる `resource_type` をもとに、適切な VM 構成を決定
- CPU コア数、メモリサイズ、ディスクサイズなどのマッピングテーブルを Server 側で管理
- Agent に対して Runner 作成コマンドを送る際に、具体的なリソース構成を指定

**実装箇所**:
- `internal/server/scheduler/scheduler.go`: リソースタイプのマッピングロジック
- `apis/proto/shoes/vz/agent/v1/agent.proto`: CreateRunnerCommand にリソース構成フィールドを追加
- Server 設定ファイル: リソースタイプ定義（CPU/メモリ/ディスク）

**設計上の考慮点**:
- リソースタイプは設定ファイルまたは環境変数で定義可能にする
- 未定義のリソースタイプが指定された場合のデフォルト値を設定
- Agent の capacity に基づいて、作成可能なリソースタイプを制限

---

### Saved State（VM 状態保存・復元）

**概要**:
SSH Ready までの起動時間をさらに短縮するため、VM の状態を保存・復元する機能（オプション機能）。

**詳細**:
- Golden Template とは別に「Warm Template（Saved State 付き）」を用意
- Fast Path（通常の Runner 作成）では、Saved State からの restore を優先
- SSH Ready 直前の状態を保存しておくことで、起動プロセスをスキップ

**実装箇所**:
- `internal/agent/vm/vm.go`: Saved State の保存・復元ロジック
- `pkg/model/config.go`: Saved State 機能の有効化フラグ
- Template ディレクトリ: `State.save` ファイルの配置

**設計上の考慮点**:
- Saved State は **VM 構成が一致**している必要がある
  - CPU / メモリ / デバイス構成を Fast Path で変更しないこと
  - リソースタイプごとに異なる Saved State を用意する必要がある
- Saved State の作成は Maintenance Path（メンテナンス時）に行う
- restore に失敗した場合は、通常起動にフォールバック

**運用方針**:
1. テンプレート VM を起動し、SSH Ready 状態まで待機
2. VM の状態を保存（`VirtualMachine.saveMachineStateTo`）
3. 保存した `State.save` ファイルを Template ディレクトリに配置
4. Runner 作成時は、State.save を clone して restore を試みる

**注意事項**:
- macOS VM の Saved State は容量が大きくなる可能性がある（APFS clone で緩和）
- ゲスト OS の状態（実行中プロセス、ネットワーク接続等）も保存されるため、互換性に注意
- Saved State の更新タイミング（OS アップデート、設定変更時等）を明確にする

---

## 改善項目

### エラーハンドリング

- より詳細なエラーメッセージ
- リカバリ処理の実装
  - VM 起動失敗時の自動リトライ
  - Agent との接続断時の再接続ロジック
- タイムアウトの最適化
  - SSH Ready 判定のタイムアウト調整
  - setup_script 実行のタイムアウト設定

### テスト

- テンプレートを使った統合テスト
- myshoes との連携テスト
- HTTP 通信のパフォーマンステスト
- 並行 Runner 作成時の負荷テスト

### ドキュメント

- VM テンプレートの作成ガイドの充実
- トラブルシューティングガイド
- パフォーマンスチューニングガイド

### メトリクス

- より詳細な起動時間メトリクス
  - clone time（APFS クローン時間）
  - boot-to-ssh time（起動から SSH Ready まで）
  - setup-script time（setup_script 実行時間）
- リソース使用率メトリクス
  - ディスク使用量（Runner ごと、テンプレートごと）
  - メモリ使用量（VM ごと）

---

## 優先順位

1. **高**: リソースタイプ管理（myshoes 連携に必須）
2. **中**: エラーハンドリング・リカバリ処理（運用安定性向上）
3. **中**: 統合テスト・myshoes 連携テスト（品質保証）
4. **低**: Saved State（パフォーマンス最適化、オプション機能）
5. **低**: 詳細メトリクス（運用改善）
