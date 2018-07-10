# `bq query` コマンドの結果を CSV に保存する例

こんな感じです。

```
bq query \
  --use_legacy_sql=false \
  --quiet \
  --max_rows=1000000 \
  --format=csv \
  "SELECT n FROM UNNEST(GENERATE_ARRAY(1, 1000)) AS n" > result.csv
```

気をつけるべきポイントは以下の通りです:

- `--max_rows` を指定しないと 100 レコードしか取れない
- `--quiet` を指定しないと最初に改行が入ってしまう

クエリは[ここ](https://qiita.com/shiozaki/items/2827e18be40335668902)から拝借しました。
