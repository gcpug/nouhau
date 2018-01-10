# Dataflow で合流する例

sink はそれぞれの出力先に適したレート制御のロジックなどが実装されているので [multi-source-multi-sink](/dataflow/example/multi-source-multi-sink/README.md) に書かれているもののように、
同じ sink に対して複数の PCollection を書き込む場合は sink を合流したほうが効率がよくなる場合がある。

合流するには `PCollectionList#apply` に Flatten を渡すなどすると良い。

```java
public static void main(String[] args) {
    DatastoreToDatastoreOptions options =
            PipelineOptionsFactory.fromArgs(args).withValidation()
                    .as(DatastoreToDatastoreOptions.class);
    String[] kinds = options.getInputKinds().split(",");
    Pipeline p = Pipeline.create(options);
    PCollectionList<Entity> pcs = PCollectionList.empty(p);
    for (String kind : kinds) {
        KindExpression kindExpression = KindExpression.newBuilder().setName(kind).build();
        Query getKindQuery = Query.newBuilder().addKind(kindExpression).build();
        PCollection<Entity> pc = p.apply(kind, DatastoreIO.v1().read().withProjectId(options.getInputProjectId()).withQuery(getKindQuery))
                .apply(new DatastoreToDatastore.EntityMigration());
        pcs = pcs.and(pc);
    }
    pcs.apply(Flatten.pCollections())
        .apply(DatastoreIO.v1().write().withProjectId(options.getOutputProjectId()));

    p.run();
}
```
