# fixture-bank — 機能完成までのTODOリスト

現状、このリポジトリは設計ドキュメント（[DESIGN.md](./docs/DESIGN.md), [DSL_SPEC.md](./docs/DSL_SPEC.md)）のみが存在し、実装コードは未着手。
[DESIGN.md](./docs/DESIGN.md) の「5. MVPスコープ」「7. ロードマップ」に基づき、機能完成までのタスクをフェーズ分割する。

## Phase 0: プロジェクト基盤

- [ ] 実装言語・ランタイムの選定（CLIとMCPサーバーを両立しやすい構成を検討。例: Node.js + TypeScript）
- [ ] プロジェクトスキャフォールディング（パッケージ管理、ディレクトリ構成、lint/format設定）
- [ ] テストフレームワークのセットアップ
- [ ] CI設定（lint/test/buildの自動実行）
- [ ] LICENSE確定（MIT想定。DESIGN.md 6. 未決事項）

## Phase 1: DSLコア（v0.1相当）

- [ ] DSLのトップレベル構造の型定義（`entity`, `count`, `seed`, `fields`, `relations`）
- [ ] YAMLパーサーの実装
- [ ] 構文検証（`type`/`generator`の組み合わせが定義済みか）
- [ ] generator実装
  - [ ] `fixed`
  - [ ] `random_int` / `random_float`
  - [ ] `sample`
  - [ ] `sequence`
  - [ ] `faker`（name, email等の主要providerから着手）
  - [ ] `uuid_v4`
  - [ ] `ref`（`parent.<field>` の解決）
  - [ ] `pool_ref`（DBクエリ結果のプールキャッシュ、空プール時 `error_type: empty_pool`）
- [ ] `unique` 制約の実装
  - [ ] `none`（デフォルト、何もしない）
  - [ ] `batch`（バッチ内一意性の保証）
  - [ ] `db`（DB存在チェック + 既定10回リトライ、失敗時 `error_type: unique_retry_exhausted`）
- [ ] `relations` の多階層生成エンジン
  - [ ] トポロジカル順（親→子）での生成制御
  - [ ] `count_per_parent`（`{fixed: N}` / `{min, max}`）の実装
  - [ ] `ref: parent.<field>` の直上親限定の解決規則
- [ ] 統一エラーモデル（`error_type` を伴うエラー返却）の実装

## Phase 2: `materialize` コマンド

- [ ] CLIエントリポイントの実装（`fixture-bank materialize`）
- [ ] オプション: `--dsl`, `--fixture`, `--count`, `--format sql|json`
- [ ] `--count` によるルートエンティティ件数の上書きロジック（子は`count_per_parent`で自動スケール）
- [ ] JSON出力の実装
- [ ] SQL出力の実装（PostgreSQL向けINSERT文生成）
- [ ] PostgreSQL接続設定（`pool_ref` / `unique: db` のクエリ発行用）

## Phase 3: Fixtureの保存・タグ管理

- [ ] ローカルファイルベースの保存形式の設計（保存先ディレクトリ構成、メタデータ形式）
- [ ] タグ形式でのFixture指定（例: `user:level50:has_premium_pass`）の解決ロジック
- [ ] Fixture一覧・検索コマンド

## Phase 4: MCPサーバー化（v0.2相当）

- [ ] `MCP_TOOLS.md` の新規作成（README/DESIGN.mdから参照されているが未作成。ツールI/Fの確定）
- [ ] `introspect_schema`: 対象DBのスキーマ調査ツール
- [ ] `draft_dsl`: DSL草案生成 + 3段階検証の実装
  - [ ] 構文検証
  - [ ] スキーマ整合検証（`introspect_schema`結果との照合）
  - [ ] DB実行検証（サンドボックスDBへの少数件試行適用、unique/FK/CHECK制約違反の検出）
- [ ] `materialize`: MCP経由での呼び出し
- [ ] `save_fixture`: 生成DSLの保存

## Phase 5: テスト・ドキュメント整備

- [ ] 各generatorのユニットテスト
- [ ] `relations`（多階層・count_per_parent）のユニットテスト
- [ ] `unique`（batch/db）のユニットテスト
- [ ] `pool_ref`（空プール含む）のユニットテスト
- [ ] PostgreSQLを用いた統合テスト（testcontainers等でのサンドボックス検証）
- [ ] README.mdの「現在のステータス」更新（設計フェーズ → 実装状況に応じて更新）

## Phase 6: 未決事項の解消（DESIGN.md 6.）

- [ ] `ref` の複数階層関連（user → order → order_item）のサポート範囲の最終決定
- [ ] `unique`違反時のリトライ戦略（サフィックス付与等）の要否判断
- [ ] DSLのバージョニング方針（スキーマ変更時の検知方法）

## Phase 7: リリース準備（v0.3〜v1.0）

- [ ] パッケージ公開（配布方法の確定）
- [ ] Zenn記事等での実践例公開、利用者フィードバック収集
- [ ] v1.0以降の検討: 複数DB対応、relations多階層対応の拡張
