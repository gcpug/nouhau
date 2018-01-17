# UDF の実行環境を調査する

ことの発端は下の BigQuery UDF で WebAssembly が呼び出せるというツイートが GCPUG Slack の `#g-bigquery_ja`で紹介されて盛り上がったところから。
https://twitter.com/francesc/status/951935862467514368

`this` に登録されているプロパティをメソッド含めて全て列挙することで、 UDF 実行環境から呼び出せるものを確認することができる。

## Standard SQL

```
#standardSQL
CREATE TEMPORARY FUNCTION getPropsRec()
RETURNS ARRAY<STRUCT<path STRING, type STRING, value STRING>>
LANGUAGE js AS """
let dump = function(obj, path) {
    let arr = [{path: path, type: typeof(obj), value: JSON.stringify(obj)}];
    if (typeof(obj) !== 'object') {
        return arr;
    }
    for (let p of Object.getOwnPropertyNames(obj)) {
        arr = arr.concat(dump(obj[p], path + '.' + p));
    }
    return arr;
};
return dump(this, '');
""";
SELECT * FROM UNNEST(getPropsRec())
```
## Legacy SQL

ドキュメントされている方法としては Legacy SQL で UDF を定義するには UDF Editor 上で定義する必要があるが、
非ドキュメント関数の `JS` で Legacy SQL 内から直接 UDF を使うことができる。
https://stackoverflow.com/questions/36207063/where-is-the-bigquery-documentation-describing-how-to-define-a-javascript-udf-fu/36208489#36208489

```
SELECT * FROM js((SELECT 1 AS x), x, "[{name: 'path', type: 'string'}, {name: 'type', type: 'string'}, {name: 'value', type: 'string'}]",
"function(row, emit) {
    let dump = function(obj, path) {
        emit({path: path, type: typeof(obj), value: JSON.stringify(obj)});
        if (typeof(obj) !== 'object') {
            return;
        }
        for (let p of Object.getOwnPropertyNames(obj)) {
            dump(obj[p], path + '.' + p);
        }
    };
    dump(this, '');
}"
)
```
