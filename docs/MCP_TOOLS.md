# fixture-bank — MCP Tools 仕様

`fixture-bank mcp` はstdio transportで動作するMCPサーバーを起動する。実装は`internal/mcpserver`、[modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)を利用している。DESIGN.mdの「エージェントによるコード・スキーマ調査 → DSL生成 → materialize」というフローのうち、前半2つ(調査・DSL生成)を支援するツール群。

```bash
$ fixture-bank mcp --db-url "$DATABASE_URL" --store-dir ./fixtures
```

- `--db-url`(省略可): `introspect_schema` / `draft_dsl`のスキーマ整合検証・DB実行検証、および`materialize`実行時の`pool_ref`/`unique: db`に使うPostgreSQL接続文字列。未指定の場合、これらの機能を使うツール呼び出しは`error_type: db_error`を返す
- `--store-dir`(省略可、既定`./fixtures`): `save_fixture`の保存先ディレクトリ

対象DBはPostgreSQLのみ(DESIGN.md ¤5「やらないこと」)。

---

## 共通のエラー表現

`materialize` / `save_fixture` / `introspect_schema` は、失敗時にMCPツールエラー(`isError: true`)として返す。エラーメッセージは`internal/ferr`の統一フォーマット `"<error_type>: <message>"` に従う(DSL_SPEC.md ¤5)。

`draft_dsl`だけは例外で、DSLが無効であること自体は「正常な検証結果」として扱うため、`isError`は立てず、`valid: false`と`stage`/`error_type`/`message`を構造化出力で返す。詳細は後述。

---

## 1. `introspect_schema`

対象PostgreSQLデータベースの`public`スキーマにあるテーブル・カラム・制約(PRIMARY KEY / UNIQUE / FOREIGN KEY)を調査する。DSLを書く前にエージェントがスキーマを把握するためのツール。

### Input

```jsonc
{
  "tables": ["user", "order"]   // 省略可。省略時はpublicスキーマの全テーブル
}
```

### Output

```jsonc
{
  "tables": [
    {
      "name": "user",
      "columns": [
        { "name": "id", "data_type": "uuid", "nullable": false, "primary_key": true, "unique": false },
        { "name": "email", "data_type": "text", "nullable": false, "primary_key": false, "unique": true },
        {
          "name": "referred_by",
          "data_type": "uuid",
          "nullable": true,
          "primary_key": false,
          "unique": false,
          "foreign_key": { "table": "user", "column": "id" }
        }
      ]
    }
  ]
}
```

### エラー

| error_type | 状況 |
|---|---|
| `db_error` | `--db-url`未設定、またはクエリ失敗 |

---

## 2. `draft_dsl`

DSL_SPEC.md ¤5に定めた3段階の検証を順に実行する。

1. **構文検証**: YAMLとして正しいか、`type`/`generator`の組み合わせが定義済みか(`internal/dsl.Parse`と同じロジック)
2. **スキーマ整合検証**: DSLが参照する`entity`/フィールド名が、`introspect_schema`と同じ調査結果に一致するか(存在しないテーブル・カラムを参照していないか)
3. **DB実行検証**: ルート1件(`count: 1`。relationsがあれば`count_per_parent`どおりに親子1セット)をトランザクション内で実際にINSERTし、unique/FK/CHECK制約違反を検出する。**このトランザクションは成否に関わらず必ずROLLBACKされ、データは残らない**

途中の段階で失敗した場合、それ以降の段階は実行しない。

### Input

```jsonc
{
  "dsl": "entity: user\ncount: 1\nfields:\n  id: {type: uuid, generator: uuid_v4}\n..."
}
```

### Output

```jsonc
// 検証成功
{ "valid": true }

// 検証失敗(どの段階でも同じ形)
{
  "valid": false,
  "stage": "schema",              // "syntax" | "schema" | "db_execution"
  "error_type": "schema_mismatch",
  "message": "user.nickname: no such column in table \"user\""
}
```

`stage`と`error_type`の対応:

| stage | error_typeの例 |
|---|---|
| `syntax` | `syntax_error` |
| `schema` | `schema_mismatch`、または(`--db-url`未設定時の)`db_error` |
| `db_execution` | `unique_retry_exhausted`、`empty_pool`、`db_error`(制約違反等) |

`schema`/`db_execution`段階は`--db-url`が設定されていないと実行できず、その場合`stage: "schema"`, `error_type: "db_error"`を返す。

---

## 3. `materialize`

DSLから任意件数のデータをJSON/SQLとして生成する。CLIの`fixture-bank materialize`と同じ`internal/materialize`エンジンを使う。

### Input

```jsonc
{
  "dsl": "entity: user\ncount: 1\n...",
  "count": 1000,     // 省略可。ルートエンティティの生成件数を上書き(DSL_SPEC.md ¤4)
  "seed": 42,        // 省略可。乱数seedを上書き
  "format": "sql"    // "json" | "sql"
}
```

### Output

```jsonc
{ "output": "INSERT INTO \"user\" (\"id\", \"level\") VALUES (...);\n..." }
```

`pool_ref` / `unique: db` を含むDSLは `--db-url` が設定されている必要がある(未設定の場合、生成中に`error_type: db_error`が返る)。

### エラー

| error_type | 状況 |
|---|---|
| `unsupported_format` | `format`が`json`/`sql`以外 |
| `syntax_error` | DSLが構文的に不正 |
| `empty_pool` | `pool_ref`のクエリ結果が0件 |
| `unique_retry_exhausted` | `unique: batch\|db`が既定10回のリトライで解決しない |
| `unknown_ref` | `ref: parent.<field>`が解決できない |
| `db_error` | DB接続未設定、またはクエリ失敗 |

---

## 4. `save_fixture`

DSLをタグ付きでローカル保存する(`internal/fixturestore`、`--store-dir`配下に`<tag>.yaml`として保存)。保存前に構文検証を行う。

### Input

```jsonc
{
  "dsl": "entity: user\ncount: 1\n...",
  "tag": "user:level50:has_premium_pass"
}
```

### Output

```jsonc
{ "tag": "user:level50:has_premium_pass" }
```

保存済みのFixtureは`fixture-bank materialize --fixture <tag>`で読み出せる(CLI側の機能。MCPツールとしての読み出し・一覧は現時点では未提供)。

### エラー

| error_type | 状況 |
|---|---|
| `syntax_error` | DSLが構文的に不正、または`tag`が空 |

---

## 今後の検討事項

- Fixtureの一覧・読み出しをMCPツールとして公開するか(現状CLIの`fixture list`/`--fixture`のみ)
- `introspect_schema`のスキーマ名指定(現状`public`固定)
- 複数DB対応(v1スコープ外。DESIGN.md ¤5)
