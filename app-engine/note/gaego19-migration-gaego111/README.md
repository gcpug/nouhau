# App Engine Standard Go 1.9 migration to Go 1.11

tag:["google-app-engine", "Go"]

App Engine Standard Go 1.9からGo 1.11への移行ではいくつか嬉しい機能が追加されています。
それは下回りがgVisorになることによって生まれた恩恵です。

gVisorがどういったものなのかは以下を確認するとよいです。

* [Java 8 ランタイム以降のサンドボックスと gVisor by @apstndb](https://docs.google.com/presentation/d/1GKkv6GAelTwieGThqnX28f6dc7fXKwDPuypRzCYT_Zk/edit#slide=id.p)
* [GCP のサーバーレスを支える gVisor by @apstndb](https://docs.google.com/presentation/d/14AAOJFsf9bSkJPA0rSAwFcn5XIcoalntCsUN8pTGO00/edit#slide=id.p)
* [gVisorとGCP by @apstndb](https://docs.google.com/presentation/d/1F6k6bBS7BOUQWl9WGEpQJDyfvd04Et_-EeIHRoQzz-Y/edit#slide=id.p)

## gVisorになって嬉しいこと

gVisorは元々のApp Engineのサンドボックス環境に存在した課題を解決するために生まれました。

### 言語の機能をそのまま使えるようにする

元々のRuntimeではGoの場合、unsafeが使えないなどの問題がありました。
ものすごく致命的というわけではないですが、一部のライブラリが使えなかったりするので、多少不便でした。

外と通信するにもApp Engine SDKの機能を利用する必要がありました。
httpを行いたければ、App Engine URLFetch APIを利用する。
Socket通信を行いたければ、App Engine Socket APIを利用する必要がありました。
gVisor版では、http clientなどがそのまま使えるようになっています。

Local FileへのWRITEも行えなかったので、一時ファイルを書き込む必要があるライブラリを使うのも困難でした。
gVisor版では `/tmp` にWRITEできるので、一時ファイルを保存できるようになりました。

### Google Cloud Spannerへの接続

従来のApp Engine SandboxではGoogle Cloud Spannerと接続することができませんでしたが、gVisor版でできるようになりました。

## Go1.11でいくつか廃止されたこと

### app.yaml includes の廃止

includesはapp.yamlに別のファイルをincludesする機能。
Deploy時に環境ごとに差異がある部分を取り込むのに地味に便利でしたが、廃止されました。
同じようなことはShellなどを駆使すればできれば可能ですが、数が多いと少々めんどうです。

## Go1.11へのマイグレーション

基本的には [公式ドキュメント通り](https://cloud.google.com/appengine/docs/standard/go111/go-differences#migrating-appengine-sdk) に行えばよい。
最低限、必要なのはapp.yamlの修正とmain packageを作成すること。
この時、ハマりやすいのが以下の項目。

### appengine.Main()

マイグレーションドキュメントのサンプルは以下のようにGoの標準機能だけで、 `http.ListenAndServe` を使うようになっている。
ただ、これはApp Engine APIから完全脱却した場合のケースで、App Engine APIを利用している場合、以下のソースの代わりに `google.golang.org/appengine` package の`appengine.Main()` を実行する。

```
// このロジックはApp Engine APIから完全脱却した場合のみ
port := os.Getenv("PORT")
if port == "" {
        port = "8080"
        log.Printf("Defaulting to port %s", port)
}

log.Printf("Listening on port %s", port)
log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
```

### script:auto

[Migrating your App Engine app from Go 1.9 to Go 1.11](https://cloud.google.com/appengine/docs/standard/go111/go-differences#migrating-appengine-sdk) に書いてなくて、地味に分かりづらいのが、app.yamlで `script: _go_app` と書いていたのを、 `script: auto` にすること。
これをしないとDeploy時にエラーになる。
[app.yaml](https://github.com/sinmetal/gcpugjp/commit/b5cdd4aa10c65719713fc31fbf4b8f63525e7162#diff-bb68be076839a38efa2ebc731942536a) のように更新する。

```
- url: /.*
  script: auto
```

### Working DirがLocalとProductionで変わる

Go 1.9までは `app.yaml` の位置がWorking Dirでしたが、Go 1.11の場合、Productionはgo.modがある位置がWorking Dirになるようになります。
しかし、 `dev_appserver.py` で実行したLocalは相変わらず `app.yaml` の位置がWorking Dirなので、噛み合いません。
そのため、 `./template/index.hml` みたいな感じでファイルをREADしていると Localでは動くけど、Deployすると `no such file or directory` になって、悲しい気持ちになります。
これの解決策はいくつかあります。
1つ目は `app.yaml` と `go.mod` の位置を同じにしてしまう方法です。
[サンプル](https://github.com/sinmetal/codelab/commit/f3eb2cacb0af05bcd733742e225fe68544fda2b3) のようにトップをmain packageにしてしまえば、OKです。
2つ目はシンボリックリンクなどを利用して、両方のPathでアクセスできるようにしてしまう方法です。

### Local開発環境の移行

元々は `goapp` というToolを利用してLocal Serverの立ち上げ、UnitTest, Deployを行っていたけど、Go1.11からは標準のやり方に寄せるようになっている。

* UnitTest `go test`
* LocalServer `dev_appserver.py`
* Deploy `gcloud app deploy`

Go 1.11から依存関係の解決に [Go Modules](https://github.com/golang/go/wiki/Modules) が使えるようになっている。
また、Deploy時に裏で [Google Cloud Build](https://cloud.google.com/cloud-build/) を利用するようになっている。
Go Modulesを利用した時にDeployの時間が長くなることがあるので、その場合は https://github.com/gcpug/nouhau/issues/87 のようなやり方をすると高速化できる。

## いつGo1.11にアップデートすべきか？

App Engine 1.9は2019年10月1日に新たにDeployをすることができなくなるので、Go 1.9のアプリケーションがある場合、すぐにGo1.11への移行をすべき。
