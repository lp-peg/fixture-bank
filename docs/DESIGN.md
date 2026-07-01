# fixture-bank — 設計ドキュメント（Draft）

## 1. 背景・課題

大規模アプリケーションの負荷試験は、以下の3つの課題を抱えやすい。

### 課題1: 試験実施作業をスケールさせづらい
シナリオが「一連のユーザー行動の再生」として直列に記述されていると、上流の機能（例:認証）がボトルネックになった場合、下流の機能のチューニングに着手できない。結果として負荷試験は少人数の専任担当に閉じがちで、分業できない。

### 課題2: 前提となるユーザーデータを作りづらい
正常系のシナリオを通すには、多くの場合「一定のリソースを保有している」といった複雑な前提状態が必要になる。この前提状態の構築を試験対象と同じ経路で行うと、初期化処理と計測対象の処理が干渉し、正確な計測ができなくなる。

### 課題3: 機能のカバー率を高めるのが難しい
課題1・2の帰結として、常に新規ユーザー相当の状態からシナリオが始まりがちで、蓄積されたデータ状態(aged state)でのパフォーマンスは検証から漏れやすい。

## 2. コンセプト: DSLによる「著者」と「量産」の分離

ユーザーが本当に欲しいのは「シナリオを通すのに必要なデータ」であり、個別具体的な値の指定ではない。一方、基盤モデルに全レコードを直接生成させると出力トークンが爆発する。

この間を埋めるのが **Fixture Bank DSL** である。

```
[ユーザー/エージェント間の自然言語での要求]
   例: "premium passを持つlevel50のユーザーで負荷試験したい"
        │
        ▼
[エージェントによるコード・スキーマ調査]
   （introspect_schema等のMCPツールを使用）
        │
        ▼
[Fixture Bank DSL の生成]  ← LLMが呼ばれるのはここだけ
   （フィールドごとの生成ルールを宣言的に記述）
        │
        ▼
[materialize: DSLをもとにデータを量産]  ← 通常のプログラム。LLM不要
   件数を指定して SQL or JSON として出力
        │
        ▼
[SQLならDBに適用 / JSONなら中身を確認]
```

エージェントは「生成ロジックの著者」であり、「データそのものの生成者」ではない。DSLという中間表現を挟むことで、1回の推論で「1件でも10万件でも同じロジックで再現性を持って量産できる仕様」を確定させる。

## 3. Fixture Bank DSL 仕様（概要。詳細は [DSL_SPEC.md](./DSL_SPEC.md) を参照）

### 3.1 基本構造

```yaml
entity: user
fields:
  id:
    type: uuid
    generator: uuid_v4
  level:
    type: int
    generator: fixed
    value: 50
  premium_pass:
    type: boolean
    generator: fixed
    value: true
  email:
    type: string
    generator: faker
    provider: email
    unique: true
relations:
  - entity: user_items
    count_per_parent: 3
    fields:
      item_id:
        generator: sample
        from: [sword_001, shield_002, potion_003]
      user_id:
        generator: ref
        ref: user.id
```

### 3.2 generatorの種類（v1スコープ）

| generator | 用途 | 例 |
|---|---|---|
| `fixed` | 固定値 | `level: 50` |
| `random_int` / `random_float` | 範囲内の乱数 | `{min: 1, max: 99}` |
| `sample` | 候補からの抽選 | `{from: [...]}` |
| `sequence` | バッチ内で連番 | `{start: 1, step: 1}` |
| `faker` | ダミーデータ(email, name等) | `{provider: "email"}` |
| `ref` | 同一バッチ内の他エンティティのフィールド参照（FK用） | `{ref: "user.id"}` |
| `uuid_v4` | UUID生成 | - |

過剰な表現力は持たせず、「負荷試験の前提データとして頻出するパターン」に絞る。カスタム式言語（DSL内DSL）は作らない。

### 3.3 materializeコマンド

```bash
$ fixture-bank materialize --dsl fixture.yaml --count 1000 --format sql
$ fixture-bank materialize --fixture user:level50:has_premium_pass --count 3 --format json
```

- `--count`: DSLに書かれた件数指定を実行時に上書きできる（少数件でJSON確認→大量件でSQL適用、という使い方を想定）
- `--format sql`: 対象DBのINSERT文として出力
- `--format json`: 生成結果をJSONとして出力（中身の目視確認、他ツールへのパイプ用途）
- 実行はDSLの解釈のみで完結し、LLM呼び出しは発生しない（決定論的・高速・無料）

## 4. 既存ツールとの違い

| | k6 / Locust / Gatling | fixture-bank |
|---|---|---|
| 得意領域 | HTTPリクエストの大量生成・スループット計測 | 前提データの生成ロジックを宣言的に定義・量産 |
| 生成ロジックの由来 | 人間が都度記述 | エージェントがコード調査の上でDSLとして著述 |
| 再現性 | シナリオスクリプト依存 | DSL + generatorが決定論的（seed値固定も可能） |
| 件数変更 | シナリオ書き換えが必要なことが多い | `--count`一つで変更可能 |

k6等を置き換えるものではなく、その前段の「複雑な前提データ構築」を担う補完ツールという位置づけは変わらない。

## 5. MVPスコープ

### やること
- [ ] DSLパーサー（YAML想定）と上記generatorの実装
- [ ] `materialize` コマンド（SQL/JSON出力、PostgreSQL対応）
- [ ] MCPサーバーとしての提供（`introspect_schema`, `draft_dsl`, `materialize`, `save_fixture` 等。詳細は[MCP_TOOLS.md](./MCP_TOOLS.md)）
- [ ] Fixtureの保存・タグ管理（ローカルファイルベース）

### やらないこと（v1では）
- 複数DB対応（まずPostgreSQLのみ）
- DSL内でのカスタム式・条件分岐（表現力を絞る）
- 実際の負荷生成（k6等の既存ツールに委譲）
- ホスティング/SaaS化

## 6. 未決事項
- `ref` で複数階層の関連（例: user → order → order_item）をどこまでサポートするか
- 一意制約(unique)違反時、DSL側でリトライ戦略（サフィックス付与等）をどう扱うか
- DSLのバージョニング（スキーマ変更時にDSLが古くなった場合の検知）
- ライセンス: MIT想定（要確認）

## 7. ロードマップ（仮）
| フェーズ | 内容 |
|---|---|
| v0.1 | DSLパーサー + materialize(SQL/JSON) + PostgreSQL対応、README公開 |
| v0.2 | MCPサーバー化（introspect_schema, draft_dsl, save_fixture） |
| v0.3 | Zenn記事での実践例公開、利用者フィードバック収集 |
| v1.0 | 複数DB対応の検討、relations多階層対応の検討 |
