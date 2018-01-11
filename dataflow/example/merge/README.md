# Dataflow で合流する例

sink はそれぞれの出力先に適したレート制御のロジックなどが実装されているので [multi-source-multi-sink](/dataflow/example/multi-source-multi-sink/README.md) に書かれているもののように、
同じ sink に対して複数の PCollection を書き込む場合は sink を合流したほうが効率がよくなる場合がある。

合流するには `PCollectionList#apply` に Flatten を渡すなどすると良い。

```java
public static void main(String[] args) {
    DatastoreToDatastoreOptions options =
            PipelineOptionsFactory.fromArgs(args).withValidation()
                    .as(DatastoreToDatastoreOptions.class);
    Pipeline p = Pipeline.create(options);
    String[] kinds = options.getInputKinds().split(",");
    List<PCollection<Entity>> pcs = new ArrayList<>();
    for (String kind : kinds) {
        KindExpression kindExpression = KindExpression.newBuilder().setName(kind).build();
        Query getKindQuery = Query.newBuilder().addKind(kindExpression).build();
        PCollection<Entity> pc = p.apply(kind, DatastoreIO.v1().read().withProjectId(options.getInputProjectId()).withQuery(getKindQuery))
                .apply(new EntityMigration());
        pcs.add(pc);
    }
    PCollectionList.of(pcs)
            .apply(Flatten.pCollections())
            .apply(DatastoreIO.v1().write().withProjectId(options.getOutputProjectId()));
    p.run();
}
```

余談だが Java 8 ネイティブな人は下のように書いても良い。 `toPCollectionList` を自前のライブラリとして用意しておくとわりとお世話になりそう。

```java
public static void main(String[] args) {
    DatastoreToDatastoreOptions options =
            PipelineOptionsFactory.fromArgs(args).withValidation()
                    .as(DatastoreToDatastoreOptions.class);
    Pipeline p = Pipeline.create(options);
    Stream.of(options.getInputKinds().split(","))
            .map(kind -> p.apply(kind, DatastoreIO.v1().read().withProjectId(options.getInputProjectId()).withQuery(buildQuery(kind))))
            .collect(toPCollectionList())
            .apply(Flatten.pCollections())
            .apply(new EntityMigration())
            .apply(DatastoreIO.v1().write().withProjectId(options.getOutputProjectId()));
    p.run();
}

private static <T> Collector<PCollection<T>, ?, PCollectionList<T>> toPCollectionList() {
    return Collectors.collectingAndThen(Collectors.toList(), PCollectionList::of);
}

private static Query buildQuery(String kind) {
    KindExpression kindExpression = KindExpression.newBuilder().setName(kind).build();
    return Query.newBuilder().addKind(kindExpression).build();
}
```
