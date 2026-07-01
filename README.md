# fixture-bank

> エージェントがコードとスキーマを調査して生成する、負荷試験用データのDSLと生成エンジン

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## これは何？

`fixture-bank` は、負荷試験に必要な「前提データ」を、エージェント(Claude Code等)が調査・設計し、決定論的に量産できるツールです。

流れはシンプルです。

1. ユーザーまたはエージェントが負荷試験シナリオを伝える（例:「premium passを持つlevel50のユーザーで負荷試験したい」）
2. エージェントがコード・スキーマを調査し、そのシナリオに必要なデータの形を把握する
3. エージェントが調査結果をもとに **Fixture Bank DSL** を生成する
4. DSLを入力に、`fixture-bank` が任意件数のデータを **SQL**（DBに直接適用）または **JSON**（中身の確認用）として生成する

ポイントは、**LLMが呼ばれるのはDSLを書く3のステップだけ**ということです。実際にデータを1件生成しようが10万件生成しようが、4のステップは通常のプログラムとして決定論的に動くため、トークンコストも実行結果のブレも発生しません。

## なぜ必要か

- 📉 **分業できない**: シナリオが直列だと、上流のボトルネックが解消するまで下流を試験できない
- 🧩 **前提データ構築が試験対象を汚染する**: 初期化処理と計測対象の処理が同じ経路を通ると正確な計測ができない
- 🕳️ **カバレッジが上がらない**: 常に新規ユーザー相当の状態からシナリオが始まりがちで、蓄積されたデータ状態(aged state)の検証が漏れる

`fixture-bank` は、前提データの生成ロジックをDSLとして切り出し、資産化することでこれらに応える。詳しい設計は [DESIGN.md](./docs/DESIGN.md)、MCPツールのI/Fは [MCP_TOOLS.md](./MCP_TOOLS.md) を参照。

## クイックイメージ

```bash
# 保存済みDSLから、SQLとして1000件生成しDBに適用
$ fixture-bank materialize --fixture user:level50:has_premium_pass --count 1000 --format sql | psql mydb

# 中身を確認したいだけならJSONで
$ fixture-bank materialize --fixture user:level50:has_premium_pass --count 3 --format json
```

## 現在のステータス

🚧 **設計フェーズ**。DSL仕様とMCPツールI/Fを固めている段階です。

## ライセンス

MIT（予定）
