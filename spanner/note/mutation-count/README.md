# mutation の数え方

tag["google-cloud-spanner"]

この記事は一度のCommitに含めることができるmutationの数の数え方について、調べた記事です。

## 前提知識

SpannerのCommitにはいくつか制限が存在しますが、その中の1つがmutationの数です。
[ドキュメント](https://cloud.google.com/spanner/quotas?hl=en#limits_for_creating_reading_updating_and_deleting_data) には `Mutations per commit (including indexes) : 20,000` と書いてあります。
ややこしいのが、このmutationというのは、 `spanner.InsertMutation` などの操作の数ではありません。
Spannerは [TransactionをRow, Column単位で行います](https://cloud.google.com/spanner/docs/transactions#rw_transaction_performance) が、このmutationの制約も同じようにRow, Columnごとに作用します。
更にIndexも絡んでくるので、なかなか計算が難しい値です。

## mutation の数え方

### INSERT

```
INSERTに含むColumnの数 + INSERTするTableに存在するIndexの数
```

``` example 1
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

INSERT INTO Measure (
  ID
)
VALUES (  
    'HogeID'
)
```

この場合、1 RowのmutationはID Columnのみがカウントされて `1` となります。

``` example 2
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

INSERT INTO Measure (
  ID,
  Col1
)
VALUES (  
    'HogeID'
    'Hello'
)
```

この場合、1 RowのmutationはID Column, Col1 Columnがカウントされて `2` となります。

``` example 3
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

CREATE INDEX MeasureCol1Index
ON Measure (
    Col1
);

INSERT INTO Measure (
  ID
)
VALUES (  
    'HogeID'
)
```

この場合、1 RowのmutationはID Column, MeasureCol1Indexがカウントされて `2` となります。
Col1に何も設定しなくても、加算されるのは、おそらくMeasureCol1IndexにNULLを入れているからだと思います。

``` example 3
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

CREATE INDEX MeasureCol1Index
ON Measure (
    Col1
);

INSERT INTO Measure (
  ID,
  Col1
)
VALUES (  
    'HogeID'
    'Hello'
)
```

この場合、1 RowのmutationはID Column, Col1 Column, MeasureCol1Indexがカウントされて `3` となります。
