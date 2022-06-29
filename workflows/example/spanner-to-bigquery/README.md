# Cloud SpannerのテーブルデータをBigQueryに定期的にコピーする

tag:["google-workflows", "google-bigquery", "google-cloud-spanner"]

## はじめに

データベースにCloud Spannerを採用しているプロジェクトでは、ユーザ利用分析や問題発生時の原因特定など、Cloud SpannerにあるデータをBigQueryにとりあえず全部持ってきていろいろ分析したいことも出てくるのではないでしょうか。
ただそのためには、一般的にSpannerからデータを取得する処理やデータをBigQueryにロードする処理を用意してApache Airflow等のワークフローエンジンで実行することになり、実行しようとするとやや腰が重いのではないかと思います。
そこでこのノウハウではフルマネージドなワークフローエンジンである[Workflows](https://cloud.google.com/workflows)を利用することで手軽に指定したSpannerのdatabaseの全テーブルをBigQueryに定期コピーする例を紹介します。

![Spanner to BigQuery](architecture.png "Spanner to BigQuery")

この例ではSpannerからのデータ取得にあたって、Googleが提供するDataflow Templateの一つである、[Cloud_Spanner_to_GCS_Avro](https://cloud.google.com/dataflow/docs/guides/templates/provided-batch#cloud_spanner_to_gcs_avro) を利用します。
このTemplateは指定したSpannerのdatabase配下の全テーブルのデータをテーブル別に区別されたAvroファイルとして保存することができます。
BigQueryではこのAvro形式のファイルのデータロードをサポートしているので、Workflowsを使ってこの公式Templateを起動してSpannerデータをAvroファイルとして出力し、テーブルごとにBigQueryにAvroファイルをロードするJobを実行するワークフローを作成しCloud Schedulerから定期実行します。

なお、この例ではGoogle公式Templateが以下の仕様を満たすことを前提とします。

* テーブルデータのAvroファイルは以下の命名規則により生成されるディレクトリ配下に出力される
  * `{指定したGCSパス}/{spannerInstanceID}-{spannerDatabaseID}-{dataflowJobID}/`
* 各テーブルのAvroファイルが上記ディレクトリ直下に以下命名規則により出力される
  * `{tableName}.avro-*`
* 全テーブル情報のリストを以下形式で含むファイルが上記ディレクトリ直下にspanner-export.jsonとして出力される
  * `{"tables": [{"name": tableName, ... }...]}`

## Workflowsファイル構成

Workflowsの設定は [spanner-to-bigquery.yaml](spanner-to-bigquery.yaml) の通りです。
steps項目で指定したstepが基本的には上から順に実行されます。
この例では以下のstepが順に実行されます。

* init: 定数や変数を定義する
* check_existing_export: 既存のSpanner export結果があるかチェックする
* export_spanner: Dataflow Templateを実行してSpannerからデータをexportする
* set_export_directory: DataflowのjobIdからexport先のディレクトリ名を生成して変数として設定する
* get_spanner_export: 出力されたspanner-export.jsonファイルからテーブル一覧を取得する
* load_bigquery: テーブル一覧からテーブルごとにAvroファイルをロードする

以下個別にstepの中身を確認していきます。

### init: 定数を定義する

最初のstepでは以下の定数を定義しています。以降のstepの定義でこれら変数を参照することができます。

* 処理を実行するGCPプロジェクトID
* データ移動対象とするSpannerのInstanceID、DatabaseID、
* Avroファイルを保存するGCSのバケット
* Dataflowを起動するリージョン、Google提供のSpannerからGCSにAvroで保存するTemplateのパス
* BigQueryの保存先Dataset名、ロケーション
* ロードが完了したテーブル名の格納変数

```yaml
main:
    params: [args]
    steps:
    - init:
        assign:
            - PROJECT_ID: ${sys.get_env("GOOGLE_CLOUD_PROJECT_ID")}
            - SPANNER_INSTANCE: xxxx
            - SPANNER_DATABASE: xxxx
            - GCS_BUCKET_BACKUP: ${PROJECT_ID+"-spanner-backup"}
            - DATAFLOW_LOCATION: us-central1
            - DATAFLOW_TEMPLATE: gs://dataflow-templates/2022-06-06-00_RC00/Cloud_Spanner_to_GCS_Avro
            - BQ_LOCATION: US
            - BQ_DATASET: xxxx
            - succeededTables: []
```


Workflowsを実行しているGCPプロジェクトは環境変数として設定されているのでsys.get_env関数を使って取得します。
この例ではSpannerやDataflow、BigQueryを同じGCPプロジェクトで動かしています。

Avroを出力するGCSのバケットの定義では先に取得したGCPプロジェクト名を使って後ろに`-spanner-backup`を付けています。
このバケットはテーブルのロード実行にあたりBigQueryのデータセットと同じリージョンに設定する必要があるので注意ください。
なおこのバケットはデータ移動に伴う一時的なファイル置き場なので、ライフサイクル設定で一定時間経過したファイルを削除するようにするとコスト削減になり良いと思われます。

最後に定義している`succeededTables`は後のBigQueryのロードしたテーブル名を集めて出力するために参照するための変数です。

また定数のかわりに実行時変数を定義してWorkflowを起動時に指定することもできます。
`params`で変数を定義しておけば、実行時にこの変数にアクセスしてパラメータやフロー制御に用いることができます。

### check_existing_export: 既存のSpanner export結果があるかチェックする

2つめのstepは、過去のSpanner export実行済みのAvroファイルを再利用するように実行時変数でexport出力先ディレクトリが設定されていた場合に、Dataflow TemplateによるSpanner exportの実行をスキップしてAvroファイルのBigQueryへのテーブルロード処理を行うように分岐処理を定義したものです。

実行時に設定した変数はargsのプロパティとして格納されます。ここではexportDirectoryという実行時変数にSpanner exportファイルが出力されたディレクトリが指定されていた場合に、後続のDataflow TemplateのJob実行をスキップするようフロー定義を記述しています。

```yaml
main:
    params: [args]
    steps:
...
    - check_existing_export:
        switch:
            - condition: ${map.get(args, "exportDirectory")!=null}
              steps:
                  - set_existing_export_directory:
                      assign:
                          - exportDirectory: ${args.exportDirectory}
                  - go_get_spanner_export:
                      next: get_spanner_export
```

### export_spanner: SpannerデータをAvroファイルとして保存するDataflow Jobを実行する

3番目のstepではGoogle公式TemplateからDataflow Jobを起動してSpannerからデータをexportしてAvroファイルを指定したGCSパス配下に保存します。

ここではdataflow Jobを起動して完了を待つ処理を[subworkflows](https://cloud.google.com/workflows/docs/reference/syntax/subworkflows)として`launch_dataflow_job_and_wait`という名前で定義してそれを呼び出しています。
処理の意味的にまとまったステップをsubworkflowsとして定義して呼び出すようにするとmainのstepsの見通しが良くなります。

```yaml
main:
    params: [args]
    steps:
...
    - export_spanner:
        call: launch_dataflow_job_and_wait
        args:
            projectId: ${PROJECT_ID}
            location: ${DATAFLOW_LOCATION}
            template: ${DATAFLOW_TEMPLATE}
            instance: ${SPANNER_INSTANCE}
            database: ${SPANNER_DATABASE}
            bucket: ${GCS_BUCKET_BACKUP}
        result: launchResult
...
launch_dataflow_job_and_wait:
    params: [projectId, location, template, instance, database, bucket]
    steps:
        - launch_dataflow_job:
            call: googleapis.dataflow.v1b3.projects.locations.templates.launch
            args:
                projectId: ${projectId}
                location: ${location}
                gcsPath: ${template}
                body:
                    jobName: spanner-backup
                    parameters:
                        instanceId: ${instance}
                        databaseId: ${database}
                        spannerProjectId: ${projectId}
                        outputDir: ${"gs://"+bucket+"/"}
                        spannerPriority: LOW
                        shouldExportTimestampAsLogicalType: "true"
                        avroTempDirectory: ${"gs://"+bucket+"/temp/"}
                validateOnly: false
            result: launchResult
        - get_dataflow_job:
            call: googleapis.dataflow.v1b3.projects.locations.jobs.get
            args:
                jobId: ${launchResult.job.id}
                location: ${location}
                projectId: ${projectId}
            result: jobResult
        - check_dataflow_job_done:
            switch:
              - condition: ${jobResult.currentState=="JOB_STATE_DONE"}
                steps:
                  - done:
                      return: ${launchResult}
              - condition: ${jobResult.currentState=="JOB_STATE_FAILED"}
                steps:
                  - failed:
                      raise: ${"Failed to launch dataflow job for spanner export"}
        - wait_for_job_completion:
            call: sys.sleep
            args:
                seconds: 30
            next: get_dataflow_job
```

`launch_dataflow_job_and_wait`の定義では`params`を定義して呼び出し元から引数として変数を渡せるようにしています。
ここではDataflowの実行リージョン、テンプレートのGCSパス、SpannerのインスタンスIDとデータベースID、Avroファイルを保存するバケット名を引数として定義しています。

subworkflowである`launch_dataflow_job_and_wait`のstepsも上から順次実行されます。
最初のstepではDataflow TemplateからDataflow Jobを起動する[Dataflow Template Launch API](https://cloud.google.com/workflows/docs/reference/googleapis/dataflow/v1b3/projects.locations.templates/launch)を指定しています。
Jobのパラメータとして、読み込むSpannerのGCPプロジェクトID、インスタンスID、データベースIDを指定しています。
またAvroファイルを出力するGCSのパスを指定しており、これらの値は引数として受け取った変数を利用しています。
その他のパラメータとして`spannerPriority`や`shouldExportTimestampAsLogicalType`を指定しています。
`spannerPriority`はSpannerからのデータ取得にあたってクエリの優先度を設定するもので、Spannerインスタンスへの負荷の影響を極力小さくするためにLOWを指定しています。
`shouldExportTimestampAsLogicalType`は、このTemplateはデフォルトではTimestamp型を文字列として出力するため、これをBigQueryがTimestamp型と認識できるようにtrueに設定しています(trueは文字列として指定)
`result`にはcallで実行した結果が`launchResult`という変数に格納することを指示しており、次以降のstepでこの変数を参照することができます。
このstepはDataflow Jobの完了を待つことなく終了し次のstepに推移します。Jobの完了を待つためにこの後のstepでこの変数を参照します。

次のstepではDataflowのJobの実行状態などを含む情報を取得する[Dataflow Job Get API](https://cloud.google.com/workflows/docs/reference/googleapis/dataflow/v1b3/projects.locations.jobs/get)を呼び出しており、Job起動時にレスポンスとして受け取ったJobIdを指定して結果はjobResultという変数に代入しています。

次のstepではjobResultの中身を確認してJobが完了していたらsubworkflowを完了して処理を抜け出すよう定義しています。
またDataflow Jobが失敗していた場合は例外を投げてworkflowの処理が失敗となるように定義しています。
Jobが終わっていなかった場合は次のstepに遷移してインターバルとして30秒待つ組み込み関数を実行しています。
30秒待ったあとは`get_dataflow_job`のstepに遷移して繰り返しJobの状態を確認するよう定義します。

### set_export_directory: Spannerのexportディレクトリを変数として設定

4番目のstepでは前のstepで完了したDataflow JobによりSpannerからexportされたファイルが格納されたディレクトリをDataflowのJob情報から組み立てて変数として設定しています。

```yaml
main:
    params: [args]
    steps:
...
    - set_export_directory:
        assign:
            - exportDirectory: ${SPANNER_INSTANCE+"-"+SPANNER_DATABASE+"-"+launchResult.job.id}
```


### get_spanner_export & download_spanner_export: Spannerのexportファイル情報を取得

5番目となるstepではDataflow Jobにより出力されたSpannerのexportファイルを確認してAvroファイルや対応するテーブル名の情報を取得します。

ここでもCloud StorageにあるファイルをダウンロードしてJSONとして取得する処理をsubworkflowとして`download_gcs_object_as_json`という名前で定義してそれを呼び出しています。

```yaml
main:
    params: [args]
    steps:
...
    - get_spanner_export:
        call: download_gcs_object_as_json
        args:
            bucket: ${GCS_BUCKET_BACKUP}
            object: ${exportDirectory+"%2F"+"spanner-export.json"}
        result: spannerExport
...
download_gcs_object_as_json:
    params: [bucket, object]
    steps:
        - get_object:
            call: googleapis.storage.v1.objects.get
            args:
                bucket: ${bucket}
                object: ${object}
            result: objectInfo
        - download_object:
            call: http.request
            args:
                url: ${objectInfo.mediaLink}
                method: GET
                auth:
                    type: OAuth2
            result: response
        - as_json:
            return: ${json.decode(response.body)}
```

`download_gcs_object_as_json`では指定されたパスのGCSのObject情報を取得して、ファイルをダウンロード、JSONファイルにデコードという順で処理を行います。

Cloud_Spanner_to_GCS_AvroはSpannerからのexport内容を記載したファイルを`{指定したGCSパス}/{spannerInstanceID}-{spannerDatabaseID}-{dataflowJobID}/spanner-export.json`に出力します。
このファイルを取得するためにまず[Cloud Storage Object Get API](https://cloud.google.com/workflows/docs/reference/googleapis/storage/v1/objects/get)でファイル内容を取得します。
引数として取得したいファイルのあるGCS bucketとobjectをパラメータとして指定します(objectはURLエンコーディングする必要があるのに注意)。

このAPIで取得した情報にはファイルの中身は含まないため、次のstepで取得したファイル情報からダウンロードリンクを取得してhttpリクエストでファイルを取得します。
その際にはworkflowsのサービスアカウントがファイルにアクセスできるようにauthでOAuth2を指定します。

httpリクエストで取得したJSONファイルの内容はbodyフィールドにバイト列として格納されているため最後のstepで組み込み関数を使ってJSONとしてデコードします。

### load_bigquery: AvroファイルからBigQueryのテーブルにデータをロードする

6番目のstepでは前のstepで取得したAvroファイルや対応するテーブル名の情報を参照して各Avroファイルを対応するBigQueryのテーブルにロードしていきます。

ここでも指定したGCSのAvroファイルからBigQueryにテーブルロードして正常完了まで待つ処理をsubworkflowとして`load_bigquery_and_wait`という名前で定義してそれを呼び出しています。
先に取得したspanner-export.jsonファイルに記載のテーブル一覧からロード処理の定義を組み立ててsubworkflowを実行し、全ての処理が完了した後にロードしたテーブル名一覧をworkflowの最終結果として出力しています。

```yaml
main:
    params: [args]
    steps:
...
    - load_bigquery:
        parallel:
            shared: [succeededTables]
            for:
                value: table
                in: ${spannerExport.tables}
                steps:
                - load_bigquery_table:
                    call: load_bigquery_and_wait
                    args:
                        projectId: ${PROJECT_ID}
                        location: ${BQ_LOCATION}
                        dataset: ${BQ_DATASET}
                        table: ${table.name}
                        sourceUri: ${"gs://"+GCS_BUCKET_BACKUP+"/"+exportDirectory+"/"+table.name+".avro*"}
                    result: bigqueryLoadJob
                - add_succeeded_tables:
                    assign:
                        - succeededTables: ${list.concat(succeededTables, bigqueryLoadJob.configuration.load.destinationTable.tableId)}
    - the_end:
        return: ${succeededTables}
...
load_bigquery_and_wait:
    params: [projectId, location, dataset, table, sourceUri]
    steps:
        - load_bigquery_table:
            call: googleapis.bigquery.v2.jobs.insert
            args:
                projectId: ${projectId}
                body:
                    configuration:
                        load:
                            createDisposition: CREATE_IF_NEEDED
                            writeDisposition: WRITE_TRUNCATE
                            destinationTable:
                                projectId: ${projectId}
                                datasetId: ${dataset}
                                tableId: ${table}
                            sourceFormat: AVRO
                            useAvroLogicalTypes: true
                            sourceUris:
                                - ${sourceUri}
            result: bigqueryLoadJob
        - get_bigquery_job:
            call: googleapis.bigquery.v2.jobs.get
            args:
                jobId: ${bigqueryLoadJob.jobReference.jobId}
                location: ${location}
                projectId: ${projectId}
            result: jobResult
        - check_bigquery_job_done:
            switch:
              - condition: ${jobResult.status.state=="DONE" AND map.get(jobResult.status, "errorResult")==null}
                steps:
                  - succeeded:
                      return: ${jobResult}
              - condition: ${jobResult.status.state=="DONE" AND map.get(jobResult.status, "errorResult")!=null AND map.get(jobResult.status.errorResult, "reason")=="backendError"}
                steps:
                  - backenderror:
                      next: load_bigquery_table
              - condition: ${jobResult.status.state=="DONE" AND map.get(jobResult.status, "errorResult")!=null}
                steps:
                  - failed:
                      raise: ${"Failed to load table "+table+" errorResult "+jobResult.status.errorResult.message}
        - wait_for_job_completion:
            call: sys.sleep
            args:
                seconds: 20
            next: get_bigquery_job
```

各テーブルのロード処理は依存関係が無いため並行に実行可能なので、[parallel句](https://cloud.google.com/workflows/docs/reference/syntax/parallel-steps)を指定しています。
このparallel句配下のstepsで定義された処理が`in`句で指定した配列ごとに並行に実行されます。
`in`句で指定した配列の中身は`value`句で定義した変数に格納されます。
先に取得したspanner-export.jsonファイルに記載のテーブル一覧を`in`句に指定してテーブルごとに`load_bigquery_and_wait`を並行実行します。
Spannerから出力されたAvroファイルは`{指定したGCSパス}/{spannerInstanceID}-{spannerDatabaseID}-{dataflowJobID}/{tableName}.avro-xxxx`として保存されています。
このテーブル名からAvroファイルのパスを組み立ててAvroファイルから同名のテーブルをBigQueryにロードします。

`load_bigquery_and_wait`では指定されたパスのGCSのAvroファイルから指定されたBigQueryのテーブルにデータをロード、正常完了を待つという順で処理を行います。
最初のstep`load_bigquery_table`では指定されたGCSパスのAvroファイルからデータをテーブルをロードするBigQueryのJobを実行する[BigQuery Jobs Insert API](https://cloud.google.com/workflows/docs/reference/googleapis/bigquery/v2/jobs/insert)を実行します。
ここではAvroのTimestampやDateなどの型をBigQueryで同じ型として取り込むために`useAvroLogicalTypes`でtrueを指定します。

次のstepでは[BigQuery Jobs Get API](https://cloud.google.com/workflows/docs/reference/googleapis/bigquery/v2/jobs/get)を使ってBigQueryのテーブルロードのJob情報を取得しています。
その次のstepではJobの状態をチェックして、正常に完了するとJob情報をsubworkflowの呼び出し元に返して処理を抜けます。
テーブルロードのJobが失敗した場合は例外を発生させて処理を異常終了させますが、backendErrorが原因の場合はリトライで正常に処理できる場合が多いため、`load_bigquery_table`に遷移してテーブルロード処理をやり直します。
処理がまだ完了していなかった場合は20秒待って`get_bigquery_job`に遷移しチェックを繰り返します。

全てのテーブルロードのJobが完了すると、workflowは正常に処理を終了します。
(一つでもテーブルロードJobが失敗した場合はworkflow全体が失敗したとみなされます)

なおここではsubworkflowが処理を正常に終えた際に取得したJob情報からテーブル名を抽出して最初に定義した`succeededTables`変数に追加していき、最後にworkflowの結果として表示しています。

## スケジュール設定

ここで定義したYAML定義からコンソール画面やgcloudコマンド、APIからWorkflowを作成します。
workflowのtriggerとしてCloud SchedulerのJobを作成・連携することができ、ここで定義した処理内容を定期実行するよう簡単に設定できます。

## おわりに

今回利用したWorkflowsは、タスクをPython等のプログラムで定義可能な既存のワークフローエンジンと比べて自由度は落ちるものの、フルマネージドで運用負担やコストが小さいため、ちょっとしたタスクに順序依存関係のあるワークフローを手軽に実行するのにとても便利だと思いました。
またWorkflowsではGCPサービスの操作は[組み込み関数として多数提供](https://cloud.google.com/workflows/docs/reference/googleapis)されており、サービスアカウントを利用した認証も連携しやすく、GCPのサービスに関するワークフローを実行するのに特に便利だと感じました。
Cloud Composerを使うにはオーバースペックに思えるようなシンプルなワークフローであればWorkflowsへの置き換えを検討しても良いかもしれません。
