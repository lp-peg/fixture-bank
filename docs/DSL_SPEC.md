# fixture-bank — DSL Specification (Draft)

## 0. 設計目標

1. **表現力を絞る**: 汎用プログラミング言語にしない。負荷試験の前提データとして頻出するパターンのみをカバーする
2. **決定論的**: 同じDSL + 同じseedなら、常に同じ構造の出力を生む
3. **多階層の関連データを自然に書ける**: `user → order → order_item` のような親子関係を無理なく表現できる
4. **実在するデータへの参照ができる**: 新規生成データだけでなく、既存の商品マスタ等への参照(FK)も表現できる

---

## 1. トップレベル構造

```yaml
entity: user          # ルートエンティティ名（DB上のテーブル/リソース名と対応）
count: 1               # デフォルト生成件数（materialize実行時の --count で上書き可能）
seed: null              # 省略可。指定すると乱数列が再現可能になる
fields:
  # フィールド定義（後述）
relations:
  # 子エンティティの定義（後述、任意の深さでネスト可能）
```

- `entity`, `fields` は必須。`count`, `seed`, `relations` は省略可
- 1ファイル = 1つのルートエンティティ + その配下の関連データ、を基本単位とする（Fixtureのタグは基本このルート単位で付ける）

---

## 2. フィールド生成（`fields`）

各フィールドは `type` と `generator` を持つ。

```yaml
fields:
  <field_name>:
    type: uuid | int | float | string | boolean | timestamp
    generator: <generator_name>
    # generator固有のパラメータ
    unique: none | batch | db     # 省略時は none
```

### 2.1 generator一覧

| generator | 説明 | パラメータ例 |
|---|---|---|
| `fixed` | 固定値 | `value: 50` |
| `random_int` | 範囲内の整数乱数 | `{min: 1, max: 99}` |
| `random_float` | 範囲内の浮動小数乱数 | `{min: 0.0, max: 1.0, precision: 2}` |
| `sample` | 候補からの抽選 | `{from: [pending, paid, shipped]}` |
| `sequence` | バッチ内で連番 | `{start: 1, step: 1}` |
| `faker` | ダミーデータ（人名・メール等） | `{provider: email}` |
| `uuid_v4` | UUID生成 | （パラメータなし） |
| `ref` | 同一生成ツリー内の他フィールド参照（主にFK用） | `{ref: parent.id}` |
| `pool_ref` | **既存DBデータ**からの抽選（後述） | `{query: "SELECT id FROM products LIMIT 200"}` |

### 2.2 `pool_ref`: 既存データへの参照

新規に生成するデータが、**すでにDBに存在するマスタデータ**（例: 商品カタログ、カテゴリ一覧）を参照しなければならないケースは多い。`pool_ref` はこれに対応する。

```yaml
fields:
  product_id:
    type: uuid
    generator: pool_ref
    query: "SELECT id FROM products WHERE status = 'active' LIMIT 200"
```

- `materialize`実行時に一度だけクエリを発行し、結果をプールとしてキャッシュ、そこから抽選する（生成件数が多くてもクエリはN回発行しない）
- プールが空の場合は `error_type: empty_pool` として即座にエラーを返す（サイレントにnullを入れたりしない）

### 2.3 `unique` の挙動

| 値 | 挙動 |
|---|---|
| `none`（既定） | 一意性を保証しない |
| `batch` | 同一`materialize`実行内でのみ一意になるよう生成する |
| `db` | DBに既存の値と衝突しないことを確認する（軽量なSELECT存在チェック） |

`db`指定時、衝突した場合は該当generatorの範囲内で再試行する（既定10回）。10回で解決しない場合は `error_type: unique_retry_exhausted` を返し、静かに諦めたり不正なデータを返したりしない。

```yaml
fields:
  email:
    type: string
    generator: faker
    provider: email
    unique: db
```

---

## 3. 関連データ（`relations`）— 多階層対応

`relations` は辞書形式で、キーが「関連名」、値がそのエンティティの定義（`fields`, `relations`をネスト可能）になる。

```yaml
entity: user
count: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}

relations:
  orders:
    entity: order
    count_per_parent: {min: 1, max: 3}   # userごとに1〜3件のorderを生成
    fields:
      id: {type: uuid, generator: uuid_v4}
      user_id: {type: uuid, generator: ref, ref: parent.id}   # 直上の親を参照
      status: {type: string, generator: sample, from: [pending, paid, shipped]}
    relations:
      items:
        entity: order_item
        count_per_parent: {fixed: 2}     # orderごとに2件のitemを生成
        fields:
          id: {type: uuid, generator: uuid_v4}
          order_id: {type: uuid, generator: ref, ref: parent.id}   # ここでのparentはorder
          product_id: {type: uuid, generator: pool_ref, query: "SELECT id FROM products LIMIT 200"}
```

### 3.1 `ref: parent.<field>` の解決規則
- `parent` は常に**直上の親エンティティ**を指す（何階層ネストしていても、相対的に一つ上を見るだけでよい）
- 祖父母以上を直接参照したい場合は、中間の親でその値を一度中継フィールドとして持たせる（例: order側に `root_user_id: {generator: ref, ref: parent.id}` を追加）ことで対応する。DSLの複雑化を避けるため、`ancestor.N` のような多段参照構文は**v1では持たない**

### 3.2 `count_per_parent` の指定方法
| 記法 | 意味 |
|---|---|
| `{fixed: 2}` | 親1件につき常に2件 |
| `{min: 1, max: 3}` | 親1件につき1〜3件のランダム |

### 3.3 生成順序
トポロジカル順（親→子）で生成する。子の`ref: parent.*`は、親がすでに確定した値を持っていることを前提にしてよい。

---

## 4. `count`の上書き

```bash
$ fixture-bank materialize --fixture user:with_orders --count 500 --format sql
```

- `--count` はルートエンティティの生成件数のみを上書きする
- 子エンティティの件数は `count_per_parent` に従って親ごとに決まる（子の総数はルート件数に連動して自動的にスケールする）

---

## 5. 検証の3段階

DSLは以下の順で検証される（`draft_dsl` MCPツール内で実施）。

1. **構文検証**: YAMLとして正しいか、`type`/`generator`の組み合わせが定義済みのものか
2. **スキーマ整合検証**: 参照している`entity`/フィールド名が`introspect_schema`の結果と一致するか（存在しないカラムを参照していないか）
3. **DB実行検証**: サンドボックスDBに少数件（既定1件、`relations`込みなら親子まとめて1セット）を試行適用し、実際の制約違反（unique, FK, CHECK）を検出する

エラーは常に `error_type` を伴って返す（詳細は [MCP_TOOLS.md](./MCP_TOOLS.md) 参照）。

---

## 6. 完全な例

```yaml
entity: user
count: 1
seed: 42
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}
  premium_pass: {type: boolean, generator: fixed, value: true}
  email: {type: string, generator: faker, provider: email, unique: db}

relations:
  orders:
    entity: order
    count_per_parent: {min: 1, max: 3}
    fields:
      id: {type: uuid, generator: uuid_v4}
      user_id: {type: uuid, generator: ref, ref: parent.id}
      status: {type: string, generator: sample, from: [pending, paid, shipped]}
    relations:
      items:
        entity: order_item
        count_per_parent: {fixed: 2}
        fields:
          id: {type: uuid, generator: uuid_v4}
          order_id: {type: uuid, generator: ref, ref: parent.id}
          product_id:
            type: uuid
            generator: pool_ref
            query: "SELECT id FROM products WHERE status = 'active' LIMIT 200"
          quantity: {type: int, generator: random_int, min: 1, max: 5}
```

`materialize --count 1000` を実行すると、user 1000件、それぞれに1〜3件のorder、各orderに2件のorder_itemが、既存商品カタログを参照しつつ生成される。

---

## 7. v1で持たない機能（意図的なスコープ外）
- 条件分岐・カスタム式（`if`, `computed`等）— DSLがプログラミング言語化するのを避けるため
- 多段祖先参照（`ancestor.2` 等）— 中継フィールドで代替可能なため
- エンティティ間の循環参照
- DSL内での外部API呼び出し

これらが本当に必要になった場合は、v1の運用実績を見てから個別に検討する。
