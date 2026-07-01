# fixture-bank — 機能完成までのTODOリスト

現状、このリポジトリは設計ドキュメント（[DESIGN.md](./docs/DESIGN.md), [DSL_SPEC.md](./docs/DSL_SPEC.md)）のみが存在し、実装コードは未着手。
[DESIGN.md](./docs/DESIGN.md) の「5. MVPスコープ」「7. ロードマップ」に基づき、機能完成までのタスクをフェーズ分割する。

## Phase 0: プロジェクト基盤 ✅

- [x] 実装言語・ランタイムの選定 → **Go 1.25**
- [x] プロジェクトスキャフォールディング（`cmd/fixture-bank`, `internal/*`, cobra CLI）
- [x] テストフレームワークのセットアップ（標準`testing`）
- [ ] CI設定（lint/test/buildの自動実行）
- [ ] LICENSE確定（MIT想定。DESIGN.md 6. 未決事項）

## Phase 1: DSLコア（v0.1相当） ✅

- [x] DSLのトップレベル構造の型定義（`entity`, `count`, `seed`, `fields`, `relations`）— `internal/dsl/types.go`
- [x] YAMLパーサーの実装 — `internal/dsl/parse.go`
- [x] 構文検証（`type`/`generator`の組み合わせが定義済みか）
- [x] generator実装 — `internal/generator/*`
  - [x] `fixed`
  - [x] `random_int` / `random_float`
  - [x] `sample`
  - [x] `sequence`
  - [x] `faker`（email, name, first_name, last_name, username, phone, address, company, word, sentence, url）
  - [x] `uuid_v4`
  - [x] `ref`（`parent.<field>` の解決）
  - [x] `pool_ref`（DBクエリ結果のプールキャッシュ、空プール時 `error_type: empty_pool`）
- [x] `unique` 制約の実装 — `internal/generator/unique.go`
  - [x] `none`（デフォルト、何もしない）
  - [x] `batch`（バッチ内一意性の保証）
  - [x] `db`（DB存在チェック + 既定10回リトライ、失敗時 `error_type: unique_retry_exhausted`）
- [x] `relations` の多階層生成エンジン — `internal/materialize/engine.go`
  - [x] トポロジカル順（親→子）での生成制御
  - [x] `count_per_parent`（`{fixed: N}` / `{min, max}`）の実装
  - [x] `ref: parent.<field>` の直上親限定の解決規則
- [x] 統一エラーモデル（`error_type` を伴うエラー返却）の実装 — `internal/ferr`

## Phase 2: `materialize` コマンド ✅

- [x] CLIエントリポイントの実装（`fixture-bank materialize`）
- [x] オプション: `--dsl`, `--fixture`, `--count`, `--seed`, `--format sql|json`, `--out`, `--db-url`
- [x] `--count` によるルートエンティティ件数の上書きロジック（子は`count_per_parent`で自動スケール）
- [x] JSON出力の実装 — `internal/output/json.go`（DSL宣言順を保持）
- [x] SQL出力の実装（PostgreSQL向けINSERT文生成）— `internal/output/sql.go`
- [x] PostgreSQL接続設定（`pool_ref` / `unique: db` のクエリ発行用）— `internal/pgdb`

## Phase 3: Fixtureの保存・タグ管理 ✅

- [x] ローカルファイルベースの保存形式の設計 — `internal/fixturestore`（タグ名 = 相対パス、`.yaml`保存）
- [x] タグ形式でのFixture指定（例: `user:level50:has_premium_pass`）の解決ロジック
- [x] Fixture一覧・検索コマンド（`fixture-bank fixture save|list`）

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

- [x] 各generatorのユニットテスト — `internal/generator/generator_test.go`
- [x] `relations`（多階層・count_per_parent）のユニットテスト — `internal/materialize/engine_test.go`
- [x] `unique`（batch/db）のユニットテスト
- [x] `pool_ref`（空プール含む）のユニットテスト
- [x] PostgreSQLを用いた統合テスト — `internal/pgdb/pgdb_test.go`（`FIXTURE_BANK_TEST_DATABASE_URL`未設定時はskip。testcontainers導入は今後の検討課題）
- [x] README.mdの「現在のステータス」更新（設計フェーズ → 実装フェーズ）

## Phase 6: 未決事項の解消（DESIGN.md 6.）

- [ ] `ref` の複数階層関連（user → order → order_item）のサポート範囲の最終決定
- [ ] `unique`違反時のリトライ戦略（サフィックス付与等）の要否判断
- [ ] DSLのバージョニング方針（スキーマ変更時の検知方法）

## Phase 7: リリース準備（v0.3〜v1.0）

- [ ] パッケージ公開（配布方法の確定）
- [ ] Zenn記事等での実践例公開、利用者フィードバック収集
- [ ] v1.0以降の検討: 複数DB対応、relations多階層対応の拡張
