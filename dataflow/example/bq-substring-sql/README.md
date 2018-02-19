# テンプレート実行時に部分文字列を指定してクエリを動的に生成する

tag:["google-cloud-dataflow","Java"]

BigQueryのSQLをテンプレート実行時に部分文字列を渡して完全に生成したいときのTipsです。
例えば日付分割されたテーブルから、日付をテンプレート実行時に範囲指定したいときに利用します。
これは[NestedValueProvider を使用する - 公式ドキュメント -> テンプレートの作成](https://cloud.google.com/dataflow/docs/templates/creating-templates?hl=ja#nestedvalueprovider-)における、

> 例 2: ユーザーが BigQuery クエリに部分文字列（特定の日付など）を指定します。変換では部分文字列を使用して完全なクエリを作成します。.get() を呼び出すと、完全なクエリが返されます。

に該当します。

```java

public interface MyOptions extends DataflowPipelineOptions {
    ValueProvider<String> getDateRange();
    void setDateRange(ValueProvier<String> value);
}

public static void main(String[] args) {

    MyOptions options =
        PipelineOptionsFactory.fromArgs(args).as(MyOptions.class);
    options.setRunnner(DataflowRunner.class);
    options.setTemplateLocation("gs://<BUCKET_NAME>/<TEMPLATE_PATH>");
    // options.set...

    Pipeline p = Pipeline.create(options);

    p.apply(BigQueryIO.readTableRows()
            .fromQuery(
                NestedValueProvider.of(
                    options.getDateRange(), new SerializableFunction<String, String>() {
                            @Override
                            public String apply(String range) {
                                String[] parts = range.split("_", 2);
                                String sql = "SELECT max FROM `bigquery-public-data.noaa_gsod.gsod*`"
                                    + " WHERE max != 9999.9 AND _TABLE_SUFFIX BETWEEN '"
                                    + parts[0] + "' AND '"
                                    + parts[1] + "' ORDER BY max DESC";
                                return sql;
                            }
                        }))
            .usingStandardSql()
            .withTemplateCompatibility()
            .withoutValidation())
        // .apply()...
```

コンパイル(テンプレートの作成)

```sh
mvn compile exec:java -Dexec.mainClass=<HOGEHOGEHOGE>
```

例えば、テンプレートの実行時に、1994年から2001年分をスキャンしたいときは、
`--parameters dateRange=1994_2001` を付与します。

```sh
gcloud dataflow jobs run dash --gcs-location gs://<BUCKET_NAME>/<TEMPLATE_PATH> --parameters dateRange=1994_2001
```
