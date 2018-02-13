# Tableの分割

tag:["google-bigquery"]

BigQueryでは日々生まれるデータを扱う場合、TableをDailyで分けていくのが定石としてあります。
現在、BigQueryではDailyで分ける方法が2つあります。
この記事ではそれぞれのやり方をメリット、デメリットをまとめていきます。

## Table Wildcard Function

Table名の末尾にYYYYMMDDを入れておけば、[TABLE_DATE_RANGE Function](https://cloud.google.com/bigquery/docs/reference/legacy-sql#table-date-range) を使って、指定した期間のTableを1クエリで扱うことができる機能。
Table自体は自分で日付で違うテーブルを作成しているだけ。
2014年春ぐらいにリリースされたような記憶。

## [Partitioned Table](https://cloud.google.com/bigquery/docs/partitioned-tables)

1つのテーブルのように見えるが、BigQueryの中でDailyに分かれているテーブルを作成する機能。
2016年春にリリースされた。
どんな機能なのかは [BigQuery の Partitioned Table 調査記録](http://qiita.com/sonots/items/0849f58627944f821beb) が参考になる。

# 違い

## Queryで読み込めるテーブル数

### Table Wildcard Function
* 1000 table

Dailyの場合、3年ぐらいになるとひっかかる感じ。

### Partitioned Table
* 無制限

Partitioned Tableはあくまで1つのテーブルとして扱われるので、クエリ実行時には特に制限はない。
ただし、 Partitonできる数が2500までなので、Dailyの場合7年弱で上限になる。

## Streaming Insert利用時にクエリに反映されるまでの時間

### Table Wildcard Function
* ストリーミングバッファに挿入された時点

ストリーミングバッファにデータが挿入された時点で、クエリに反映されます。

### Partitioned Table
* テーブルに挿入された時点 or デコレータで指定したPartition

ストリーミングバッファに挿入された段階では、 `_PARTITIONTIME` の値がNULLになっているため、テーブルに挿入されないとクエリで `_PARTITIONTIME` を利用している場合、反映されない。
ただし、デコレータを指定すれば、任意のPartitionに挿入することができる。(任意のデコレータを指定した場合に、Bigtableに挿入された時点で、クエリに反映されるのかはまだ検証していない。)

## Schemaの変更

### Table Wildcard Function
* 比較的楽

日々のテーブルは、それぞれ独立しているので、日が変わる時に変えるのはそんなに難しくない。
ただ、過去の期間を含めて、クエリを投げる時に、全てのテーブルに指定したカラムがない場合はエラーとなるので、クエリ側で工夫が必要。

### Partitioned Table
* 結構つらい

あくまで1つのテーブルなので、途中でカラムを追加することはできるが、カラムを削ったりするのはなかなか難しい。

## パフォーマンス

* Partitioned Table > Table Wildcard Function

Table Wildcard Functionは後ろではUNIONをしているっぽいので、テーブル数が多いと結構パフォーマンスが落ちる

例えば、以下のケースだと、10倍の差が出る。
[GCPUG](https://gcpug.jp) のApplication Log 1年分のデータに対してクエリを実行した。

* Table Wildcard Function 20sec
* Partitioned Table 2sec

### Table Wildcard Function

``` sql
SELECT
  ClientType,
  SUM(COUNT) AS COUNT
FROM (
  SELECT
    CASE 
      WHEN protoPayload.userAgent CONTAINS "Android" THEN "Android" 
      WHEN protoPayload.userAgent CONTAINS "iPhone" THEN "iPhone" 
      WHEN protoPayload.userAgent CONTAINS "Windows" THEN "Windows" 
      WHEN protoPayload.userAgent CONTAINS "Mac OS X" THEN "Mac OS X" 
      WHEN protoPayload.userAgent CONTAINS "Linux" THEN "Linux" 
      ELSE "Other" 
    END AS ClientType,
    COUNT(protoPayload.userAgent) AS COUNT
  FROM
    TABLE_DATE_RANGE([gcp-ug:gaelog_from_bqstreaming.appengine_googleapis_com_request_log_], TIMESTAMP('2016-01-01'), TIMESTAMP('2016-12-31'))
  WHERE
    protoPayload.resource = "/"
  GROUP BY
    protoPayload.userAgent,
    ClientType )
GROUP BY
  ClientType
ORDER BY
  COUNT DESC
```

### Partitioned Table

``` sql
SELECT
  ClientType,
  SUM(COUNT) AS COUNT
FROM (
  SELECT
    CASE
      WHEN protoPayload.userAgent CONTAINS "Android" THEN "Android"
      WHEN protoPayload.userAgent CONTAINS "iPhone" THEN "iPhone"
      WHEN protoPayload.userAgent CONTAINS "Windows" THEN "Windows"
      WHEN protoPayload.userAgent CONTAINS "Mac OS X" THEN "Mac OS X"
      WHEN protoPayload.userAgent CONTAINS "Linux" THEN "Linux"
      ELSE "Other"
    END AS ClientType,
    COUNT(protoPayload.userAgent) AS COUNT
  FROM
    [hoge.log]
  WHERE
    protoPayload.resource = "/"
    AND _PARTITIONTIME BETWEEN TIMESTAMP('2016-01-01') AND TIMESTAMP('2016-12-31')
  GROUP BY
    protoPayload.userAgent,
    ClientType )
GROUP BY
  ClientType
ORDER BY
  COUNT DESC
```

## 料金

基本的には変わらないが、以下のケースの場合、差が出る

* クエリで読み込む1テーブルのデータ容量が10MBを下回っている

クエリ実行時に読み込んだテーブル毎の最低課金単位は10MB ([クエリのオンデマンド料金](https://cloud.google.com/bigquery/pricing#on_demand_pricing))

```
料金は MB 単位のデータ処理容量（端数は四捨五入）で決まります。
クエリが参照するテーブルあたりのデータ処理容量は最低 10 MB で、クエリあたりのデータ処理容量は最低 10 MB です。
```

テーブル毎に計算されるので、1日のテーブルが小さく、たくさんのテーブルを読み込む場合に結構差が出る。
例えば、GCPUGの場合、1日のアクセス数が少なく、クエリで読み込むカラム数が少ないため、以下のようになります。

### Table Wildcard Function

* Bytes Processed : 183 MB
* Bytes Billed : 3.57 GB

1テーブルのデータ容量は少ないので、Bytes Processed(クエリ実行前に予測された値)は非常に小さくなっていますが、Bytes Billed(実際の課金額)は大きな値になっています。
これはテーブル数 * 10MBが最低課金額になるためです。(3.65GBじゃないのは、数日ログが無い日があるため)

``` sql
SELECT
  ClientType,
  SUM(COUNT) AS COUNT
FROM (
  SELECT
    CASE 
      WHEN protoPayload.userAgent CONTAINS "Android" THEN "Android" 
      WHEN protoPayload.userAgent CONTAINS "iPhone" THEN "iPhone" 
      WHEN protoPayload.userAgent CONTAINS "Windows" THEN "Windows" 
      WHEN protoPayload.userAgent CONTAINS "Mac OS X" THEN "Mac OS X" 
      WHEN protoPayload.userAgent CONTAINS "Linux" THEN "Linux" 
      ELSE "Other" 
    END AS ClientType,
    COUNT(protoPayload.userAgent) AS COUNT
  FROM
    TABLE_DATE_RANGE([gcp-ug:gaelog_from_bqstreaming.appengine_googleapis_com_request_log_], TIMESTAMP('2016-01-01'), TIMESTAMP('2016-12-31'))
  WHERE
    protoPayload.resource = "/"
  GROUP BY
    protoPayload.userAgent,
    ClientType )
GROUP BY
  ClientType
ORDER BY
  COUNT DESC
```

### Partitioned Table

* Bytes Processed : 1.52 GB
* Bytes Billed : 1.53 GB

1テーブルの中に全てのデータが収まっているので、Bytes ProcessedとBytes Billedがほぼ同じ値です。

``` sql
SELECT
  ClientType,
  SUM(COUNT) AS COUNT
FROM (
  SELECT
    CASE
      WHEN protoPayload.userAgent CONTAINS "Android" THEN "Android"
      WHEN protoPayload.userAgent CONTAINS "iPhone" THEN "iPhone"
      WHEN protoPayload.userAgent CONTAINS "Windows" THEN "Windows"
      WHEN protoPayload.userAgent CONTAINS "Mac OS X" THEN "Mac OS X"
      WHEN protoPayload.userAgent CONTAINS "Linux" THEN "Linux"
      ELSE "Other"
    END AS ClientType,
    COUNT(protoPayload.userAgent) AS COUNT
  FROM
    [hoge.log]
  WHERE
    protoPayload.resource = "/"
    AND _PARTITIONTIME BETWEEN TIMESTAMP('2016-01-01') AND TIMESTAMP('2016-12-31')
  GROUP BY
    protoPayload.userAgent,
    ClientType )
GROUP BY
  ClientType
ORDER BY
  COUNT DESC
```

# Resources

* [BigQuery の Partitioned Table 調査記録](http://qiita.com/sonots/items/0849f58627944f821beb)
* [BigQueryのクエリー料金の些細な話](http://qiita.com/kunit/items/48ed7ad7302762de8cb3)
* [Google BigQuery でヒストリカルデータ保存の料金を半分に、 クエリの速度を 10 倍に](https://cloudplatform-jp.googleblog.com/2016/04/google-bigquery-10.html)
