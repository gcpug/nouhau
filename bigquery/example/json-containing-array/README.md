# 文字列としてのJSONにARRAY<STRUCT>が含まれている時に、中身を抜き出す

tag:["bigquery"]

## 問題

`'{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}] }}'` のようなJSONがある時にfriendsの中身を抜き出すには以下のようなSQLを実行する。
しかし、この場合、配列の添字を明示的に指定する必要があるので、配列のサイズが分からない場合が困る。
なんとか、配列の数を気にすること無く、中身をすべて抜き出したい

``` sql
#standardSQL
SELECT JSON_EXTRACT_SCALAR('{"userno":4,"friends":[{"userno":2, "nickname":"hogehoge"}, {"userno":3, "nickname":"fugafuga"}] }}', "$.friends[0].nickname") as nickname
```

## 解決策

### SQLで頑張って文字列を分解していく

#### メリット

* SQLで完結している

#### デメリット

* くっそSQLが見づらい

``` sql
#standardSQL
WITH
  t AS (
  SELECT
    '{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}}' AS j)
SELECT
  userno,
  JSON_EXTRACT_SCALAR(friend,
    "$.userno") AS friend_userno,
  JSON_EXTRACT_SCALAR(friend,
    "$.nickname") AS friend_nickname
FROM (
  SELECT
    userno,
    CONCAT("{",REGEXP_REPLACE(friend, "^\\[\\{|\\}\\]$",""), "}") AS friend
  FROM (
    SELECT
      userno,
      friend
    FROM (
      SELECT
        userno,
        SPLIT( friends, "},{") AS friends
      FROM (
        SELECT
          JSON_EXTRACT_SCALAR(t.j,
            "$.userno") AS userno,
          JSON_EXTRACT(t.j,
            "$.friends") AS friends
        FROM
          t ))
    CROSS JOIN
      UNNEST(friends) AS friend))
```

ややこしいSQLなので、順を追って実行してみよう

#### Query 分解 1

まずは、単純にJSON Functionを利用して、usernoとfriendsをJSONから抜き出します。

``` sql
#standardSQL
WITH
  t AS (
  SELECT
    '{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}}' AS j)
SELECT
  JSON_EXTRACT_SCALAR(t.j,
    "$.userno") AS userno,
  JSON_EXTRACT(t.j,
    "$.friends") AS friends
FROM
  t
```

``` json
[
  {
    "userno": "12345",
    "friends": "[{\"userno\":1,\"nickname\":\"hogehoge\"},{\"userno\":2,\"nickname\":\"fugafuga\"}]"
  }
]
```

#### Query 分解 2

friendsの中のARRAY JSONをSPLITで配列に解体します。

``` sql
#standardSQL
WITH
  t AS (
  SELECT
    '{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}}' AS j)
SELECT
  userno,
  SPLIT( friends, "},{") AS friends
FROM (
  SELECT
    JSON_EXTRACT_SCALAR(t.j,
      "$.userno") AS userno,
    JSON_EXTRACT(t.j,
      "$.friends") AS friends
  FROM
    t )
```

``` json
[
  {
    "userno": "12345",
    "friends": [
      "[{\"userno\":1,\"nickname\":\"hogehoge\"",
      "\"userno\":1,\"nickname\":\"fugafuga\"}]"
    ]
  }
]
```

#### Query 分解 3

解体したfriendsのARRAYがJSONとしてINVALIDなので、編集したいですが、ARRAYのままでは関数を適用できないので、JOINします。
JOINしてFLATになり、行が2行になりました。
ただ、friendの中身は相変わらず、ただの文字列です。

``` sql
#standardSQL
WITH
  t AS (
  SELECT
    '{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}}' AS j)
SELECT
  userno,
  friend
FROM (
  SELECT
    userno,
    SPLIT( friends, "},{") AS friends
  FROM (
    SELECT
      JSON_EXTRACT_SCALAR(t.j,
        "$.userno") AS userno,
      JSON_EXTRACT(t.j,
        "$.friends") AS friends
    FROM
      t ))
CROSS JOIN
  UNNEST(friends) AS friend
```

```
[
  {
    "userno": "12345",
    "friend": "[{\"userno\":1,\"nickname\":\"hogehoge\""
  },
  {
    "userno": "12345",
    "friend": "\"userno\":1,\"nickname\":\"fugafuga\"}]"
  }
]
```

#### Query 分解 4

friendsの文字列がJSONとしてINVALIDなのを、REGEXP_REPLACE, CONCATを利用して、整形します。
ここまで来れば、勝ったも同然ですね。
後は最初のクエリの通り、friendsに対して JSON Functionをかければ完了です。

``` sql
#standardSQL
WITH
  t AS (
  SELECT
    '{"userno":12345,"friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}}' AS j)
SELECT
  userno,
  CONCAT("{",REGEXP_REPLACE(friend, "^\\[\\{|\\}\\]$",""), "}") AS friend
FROM (
  SELECT
    userno,
    friend
  FROM (
    SELECT
      userno,
      SPLIT( friends, "},{") AS friends
    FROM (
      SELECT
        JSON_EXTRACT_SCALAR(t.j,
          "$.userno") AS userno,
        JSON_EXTRACT(t.j,
          "$.friends") AS friends
      FROM
        t ))
  CROSS JOIN
    UNNEST(friends) AS friend)
```

``` json
[
  {
    "userno": "12345",
    "friend": "{\"userno\":1,\"nickname\":\"hogehoge\"}"
  },
  {
    "userno": "12345",
    "friend": "{\"userno\":1,\"nickname\":\"fugafuga\"}"
  }
]
```

### JavaScript UDFを利用する

#### メリット

* 割りときれいに見える

#### デメリット

* JavaScript UDFはProjectごとに同時実行が6つなので、Quotaにひっかかりやすい https://cloud.google.com/bigquery/quotas?hl=en#query_jobs

```
#standardSQL
CREATE TEMPORARY FUNCTION
  parse_logdata(logdata STRING)
  RETURNS STRUCT<userno INT64,
  friends ARRAY<STRUCT<userno INT64,
  nickname STRING>>>
  LANGUAGE js AS """
  return JSON.parse(logdata);
""";
WITH
  LOGDATA_TABLE AS (
  SELECT
    '{  "date": "2015-08-31",  "time": "00:00:00",  "type": "RESPONSE",  "userno": 12345,  "friends":[{"userno":1, "nickname":"hogehoge"}, {"userno":1, "nickname":"fugafuga"}]}' AS logdata
  UNION ALL
  SELECT
    '{  "date": "2015-08-31",  "time": "00:00:00",  "type": "RESPONSE",  "userno": 12346,  "friends":[{"userno":1, "nickname":"foobar"}, {"userno":1, "nickname":"piyopiyo"}]}')
SELECT
  logdata.userno AS userno,
  friends.userno AS friend_no,
  nickname
FROM (
  SELECT
    parse_logdata(logdata).*
  FROM
    LOGDATA_TABLE) AS logdata
INNER JOIN
  UNNEST(logdata.friends) AS friends
```

## 関連情報

* [BigQueryでJSON文字列を保存して配列になっている値を集計したい場合のやり方
](https://ja.stackoverflow.com/q/15170/4361)
* [How to get all values of an attribute of json array with jsonpath bigquery in bigquery? Asterisk operator not supported.](https://stackoverflow.com/q/28719880/1393813)
* [Add function json_extract_array: string -> array<string>](https://issuetracker.google.com/issues/63716683)
* [Add function json_extract_map: string -> map<string, scalar_type>](https://issuetracker.google.com/issues/65488665)