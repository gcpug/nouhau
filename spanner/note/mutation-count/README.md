# mutation の数え方

tag["google-cloud-spanner"]

この記事は一度のCommitに含めることができるmutationの数の数え方について、調べた記事です。

## 前提知識

SpannerのCommitにはいくつか制限が存在しますが、その中の1つがmutationの数です。
[ドキュメント](https://cloud.google.com/spanner/quotas?hl=en#limits_for_creating_reading_updating_and_deleting_data) には `Mutations per commit (including indexes) : 20,000` と書いてあります。
ややこしいのが、このmutationというのは、 `spanner.InsertMutation` などの操作の数ではありません。
Spannerは [TransactionをRow, Column単位で行います](https://cloud.google.com/spanner/docs/transactions#rw_transaction_performance) が、このmutationの制約も同じようにRow, Columnごとに作用します。
更にIndexも絡んでくるので、なかなか計算が難しい値です。

## INSERT

```
INSERTに含むColumnの数 + INSERTするTableに存在するIndexの数
```

INSERTは、INSERTに含むColumnの数と対象にしたTableに存在するIndexの数でmutationが決まります。
注意すべき点はIndexが参照しているColumnに値を設定しなくても、Indexはmutationに数えられる点です。
これはNULLの値をIndexに入れる必要があるからだと思われます。

Indexは複数のColumnを持つComposite Indexであっても、Storing Indexであっても、数え方は変わりません。

## Example

### PKのみ設定した最もシンプルなケース

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

### PK以外のColumnも設定するケース

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

### Indexが存在し、値を設定しないケース

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
Col1に何も設定しなくても、加算されるのは、MeasureCol1IndexにNULLを入れているからだと思います。

### Indexが存在し、値を設定するケース

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

## UPDATE

```
UPDATEに含むColumnの数 + UPDATEするColumnを含むIndexの数 * 2
```

少しややこしいですが、UPDATEの時はIndexが存在する場合のmutationの数え方が、INSERTとは異なります。
INSERTの時は初めてRowが追加されるので、必ずIndexにRowが入りますが、UPDATEの時は更新対象になっているかどうかで変わります。
もう一つ大きなINSERTとの違いはIndexはmutationの数が * 2になることです。
これはIndexに対して古いRowのDELETEと新しいRowのINSERTを行っているからではないかと考えられます。
Indexは元TableのColumnの値をPKとして保持していると考えられ、PKは更新はできないので、DELETEとINSERTを使って洗い替えを行っていると思われます。

Indexは複数のColumnを持つComposite Indexであっても、Storing Indexであっても、数え方には変わりません。

## Example

### Indexを含まないColumnを更新するケース

``` example 1
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

UPDATE Measure 
  SET Col1 = "a" 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 RowのmutationはID, Col1がカウントされて `2` となります。

### Indexを含むColumnを更新するケース

``` example 2
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

UPDATE Measure 
  SET Col1 = "a" 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 RowのmutationはID, Col1, MeasureCol1Indexがカウントされて、MeasureCol1Indexは * 2 されるので、 `4` となります。

### Composite Indexを含むColumnの一部を更新するケース

``` example 2
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
    WithCompositeIndex1 STRING(MAX),
    WithCompositeIndex2 STRING(MAX),
) PRIMARY KEY (ID);

CREATE INDEX MeasureCompositeIndex
ON MeasureCompositeIndex (
    WithCompositeIndex1,
    WithCompositeIndex2 DESC
);

UPDATE Measure 
  SET WithCompositeIndex1 = "a" 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 RowのmutationはID, WithCompositeIndex1, MeasureCompositeIndex がカウントされて、MeasureCompositeIndex は * 2 されるので、 `4` となります。

### Composite Indexを含むColumnのすべてを更新するケース

``` example 2
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
    WithCompositeIndex1 STRING(MAX),
    WithCompositeIndex2 STRING(MAX),
) PRIMARY KEY (ID);

CREATE INDEX MeasureCompositeIndex
ON MeasureCompositeIndex (
    WithCompositeIndex1,
    WithCompositeIndex2 DESC
);

UPDATE Measure 
  SET WithCompositeIndex1 = "a", WithCompositeIndex2 = "b"
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 RowのmutationはID, WithCompositeIndex1, WithCompositeIndex2, MeasureCompositeIndex がカウントされて、MeasureCompositeIndex は * 2 されるので、 `5` となります。

## DELETE

```
1 + DELETEするTableに存在するIndexの数
```

DELETEはINSERT, UPDATEに比べてmutationの数がぐっと少なくなります。
DELETE時はColumnをそもそも指定しないので、特にColumnの数というのは無くて、単純にRowに対して1になり、後はIndexの数を加算するだけです。

少し他と異なるのが、Interleaveを設定して、DELETE CASCADEを使用するケースです。
親を削除した場合、子どもも削除されますが、mutationとしては子どものTableはカウントされません。
しかし、子どものTableにInterleaveしていないIndexがある場合、そのIndexの数がmutationにカウントされます。

### PKを指定して削除する最もシンプルなケース

``` example 1
CREATE TABLE Measure (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

DELETE Measure 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 Rowのmutationは `1` となります。

### Indexを持つTableを削除するケース

``` example 2
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

DELETE Measure 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 Rowのmutationは TableとMeasureCol1Indexがカウントされて、 `2` となります。

### Interleave DELETE CASCADEを設定していて、Indexがないケース

``` example 3
CREATE TABLE MeasureParent (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

CREATE TABLE MeasureChild (
    ID STRING(MAX) NOT NULL,
    ChildID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID, ChildID),
  INTERLEAVE IN PARENT MeasureParent ON DELETE CASCADE;

DELETE MeasureParent 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 Rowのmutationは `1` となります。

### Interleave DELETE CASCADEを設定していて、Indexがあるケース

``` example 4
CREATE TABLE MeasureParent (
    ID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID);

CREATE TABLE MeasureChild (
    ID STRING(MAX) NOT NULL,
    ChildID STRING(MAX) NOT NULL,
    Col1 STRING(MAX),
    Col2 STRING(MAX),
    Col3 STRING(MAX),
) PRIMARY KEY (ID, ChildID),
  INTERLEAVE IN PARENT MeasureParent ON DELETE CASCADE;

CREATE INDEX MeasureChildWithIndexWithIndex1_1
ON MeasureChildWithIndex (
    WithIndex1
);

DELETE MeasureParent 
  WHERE ID = "000008f7-b5a3-4ada-8852-f5bf63f9e8ef"
```

この場合、1 RowのmutationはMeasureParent, MeasureChildWithIndexWithIndex1_1がカウントされて、 `2` となります。

## おまけ : どうやって調べたか？

SpannerのAPIはmutationの数を教えてくれません。
唯一教えてくれるのが、20,000を超えた時のエラーです。
そのため、mutationの数の仮説を立て、20,000, 20,001になるように実際にSpannerで実行して、仮説がある程度正しいかどうかを検証するという方法で数えました。
誤りや新しいパターンを見つけたら、ぜひContributeをお願いします。