# App Engine Standard Go 1.9 migration to Go 1.11に最低限必要なこと

tag:["google-app-engine", "Go"]

このドキュメントではGo 1.9のアプリケーションをGo 1.11に移行するために最低限必要なことと注意点について記します。

まずチェックすべきは [公式ドキュメント](https://cloud.google.com/appengine/docs/standard/go111/go-differences#migrating-appengine-sdk) です。
ただ、公式ドキュメントでは微妙なニュアンスになっているものや欠けているものがあるので、そこをフォローしていきます。

## Go1.11へのマイグレーション

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
これの解決策は3つあります。
1つ目は `app.yaml` と `go.mod` の位置を同じにしてしまう方法です。
[サンプル](https://github.com/sinmetal/codelab/commit/f3eb2cacb0af05bcd733742e225fe68544fda2b3) のようにトップをmain packageにしてしまえば、OKです。
2つ目は `app.yaml` に [main packageの位置を指定できる機能](https://cloud.google.com/appengine/docs/standard/go111/config/appref?hl=en#main) が追加されているので、こいつを使います。
3つ目はシンボリックリンクなどを利用して、両方のPathでアクセスできるようにしてしまう方法です。

### Local開発環境の移行

元々は `goapp` というToolを利用してLocal Serverの立ち上げ、UnitTest, Deployを行っていたけど、Go1.11からは標準のやり方に寄せるようになっている。

* UnitTest `go test`
* LocalServer `dev_appserver.py`
* Deploy `gcloud app deploy`

Go 1.11から依存関係の解決に [Go Modules](https://github.com/golang/go/wiki/Modules) が使えるようになっている。
また、Deploy時に裏で [Google Cloud Build](https://cloud.google.com/cloud-build/) を利用するようになっている。
Go Modulesを利用した時にDeployの時間が長くなることがあるので、その場合は https://github.com/gcpug/nouhau/issues/87 のようなやり方をすると高速化できる。

## Go1.11でConfig周りで廃止されたもの

### app.yaml からいくつかの機能が廃止された

いくつか廃止されてるが、使ってる人が多そうなのは以下の機能

#### includes

includesはapp.yamlに別のファイルをincludesする機能。
Deploy時に環境ごとに差異がある部分を取り込むのに地味に便利でしたが、廃止されました。
同じようなことはShellなどを駆使すればできれば可能ですが、数が多いと少々めんどうです。

#### skip_files

`gcloud app deploy` でdeployするようになるので、 [.gcloudignore](https://cloud.google.com/sdk/gcloud/reference/topic/gcloudignore) を使うように変更された。

## なんか書いてあるけど、とりあえずやらなくていいやつ

公式ドキュメントの中でOptionalと書いてある部分だが、App Engine APIはGo 1.11でそのまま使えるようになっている。
そのため、 `login:admin` や App Engine Search API, App Engine Memcacheなどを移行する必要はない。