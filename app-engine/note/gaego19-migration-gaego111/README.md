# App Engine Standard Go 1.9 migration to Go 1.11

tag:["google-app-engine", "Go"]

App Engine Standard Go 1.9からGo 1.11へのマイグレーションは、単純なGoのアップデートだけでなく、ランタイム自体に大きな変更が入っています。
そのため、単純にGo 1.11を使うようにしただけではDeployすることができず、ある程度の作業が必要です。
この記事はその作業はどのぐらいのものなのかと、Google App Engine Standard 2nd Generationと呼ばれる新たなランタイムを使うと何が変わるのかについて、そして、いつアップデートすべきか？について記します。

Google App Engine Standard 2nd Generationがどういうものなのかは以下を確認するとよいです。

* [Java 8 ランタイム以降のサンドボックスと gVisor by @apstndb](https://docs.google.com/presentation/d/1GKkv6GAelTwieGThqnX28f6dc7fXKwDPuypRzCYT_Zk/edit#slide=id.p)
* [GCP のサーバーレスを支える gVisor by @apstndb](https://docs.google.com/presentation/d/14AAOJFsf9bSkJPA0rSAwFcn5XIcoalntCsUN8pTGO00/edit#slide=id.p)
* [gVisorとGCP by @apstndb](https://docs.google.com/presentation/d/1F6k6bBS7BOUQWl9WGEpQJDyfvd04Et_-EeIHRoQzz-Y/edit#slide=id.p)

## 2nd Generationになって嬉しいこと

2nd Generationは1st Generationを以下を解決するために生まれました。

### 言語の機能をそのまま使えるようにする

1st GenerationではGoの場合、unsafeが使えないなどの問題がありました。
ものすごく致命的というわけではないですが、一部のライブラリが使えなかったりするので、多少不便でした。

外と通信するにもApp Engine SDKの機能を利用する必要がありました。
httpを行いたければ、App Engine URLFetch APIを利用する。
Socket通信を行いたければ、App Engine Socket APIを利用する必要がありました。
2nd Generationでは、http clientなどがそのまま使えるようになっています。

Local FileへのWRITEも行えなかったので、一時ファイルを書き込む必要があるライブラリを使うのも困難でした。
2nd Generationでは `/tmp` にWRITEできるので、一時ファイルを保存できるようになりました。

### App Engine APIからの脱却

App Engineは単純にHTTP Serverとして存在しているわけではなく、非常に多くの機能を持っています。
Log, Datastore, Memcache, Users, Mail, Search, TaskQueue, Image, Blobstoreなどです。
これらはGoogle Cloud Platformが登場する以前から存在し、App Engine SDKとしてClientが提供されていました。
便利な機能たちではありますが、App Engine専用の存在なので、強力なロックインを招きます。

2nd GenerationではなるべくApp Engine packageを使わない方向に進んでいます。
1.9まではログを出力するにも `google.golang.org/appengine/log` pakcageを利用する必要がありましたが、1.11では単純に標準出力に出力するだけでログが出力できます。

urlfetch, socketも使う必要がなくなりました。
これらはGoogle Cloud PubSubやGoogle Cloud StorageなどGCPのサービスを利用する時も必要で、更にQuotaがあるというものだったので、非常に不便でしたが、解決しました。

### Google Cloud Spannerへの接続

App Engine 1st GenerationはGoogle Cloud Spannerと接続することができませんでしたが、2nd Generationでできるようになりました。

## 2nd Generationになって悲しいこと

### `login:required` , `login:admin` の廃止

Users Serviceの機能として、 `app.yaml` に記述すれば、アクセス制限ができたこの機能が廃止されました。
Go 1.11ではすでに動かないので、なんらかの方法で実装し直す必要があります。
ただ、2018/11/25時点ではこの機能を完全に代替えするGCPの機能はありません。
似たような機能として [Cloud Identity-Aware Proxy](https://cloud.google.com/iap/) があるのですが、Users Serviceとまったく同じことはできません。
できないこととして、 `default` serviceは認証なしでOKで、 `/mypage` は `login:requreid` で、 `admin` serviceは `login:admin` みたいなことができません。
Serviceごとに認証可能なメンバーを変更するということはできるのですが、 `Reqeust Pathで変更する` のと、 `いずれかのServiceは認証なし` というのができません。
公式ドキュメントのMigration先はFirebase Authなどを使って、頑張れ！となっているのですが、結構頑張らないとできない気がします。

### Request Logがまとまらない

1st GenerationはRequestごとにApplication LogがきれいにStackdriver Loggingの1Entityにまとまるようになっていました。
しかし、2nd GenerationではLog出力するとすべて別のEntityとして出力されるので、複数リクエストが来ている状態だと、見るのがなかなか大変です。
出力したログメッセージ以外の部分にリクエスト情報などたくさんの情報が付与されるので、ログの容量が大きくなるのも嬉しくない点です。

### app.yaml includes の廃止

includesはapp.yamlに別のファイルをincludesする機能。
Deploy時に環境ごとに差異がある部分を取り込むのに地味に便利でしたが、廃止されました。
同じようなことはShellなどを駆使すればできれば可能ですが、数が多いと少々めんどうです。

### 脱却しきれないApp Engine API

Go 1.11ではApp Engine APIたちが基本的にサポートされたままなので、必ずしもApp Engine APIを完全脱却する必要はありません。
ただ、完全脱却できるとUnitTestがやりやすくなったりするので、挑戦したくなるのが人情です。
しかし、現状、完全脱却は難しい現状があります。
urlfetch, socketのように完全に脱却できたものもありますが、移行先のGCPの機能がないものが多いです。

#### App Engine APIの移行先

* Users -> Identity-Aware Proxy?
* Memcache -> Cloud Memorystore for Redis
* Datastore -> Cloud Datastore
* Search -> ...?
* Mail -> SendGrid?
* TaskQueue -> Cloud Tasks
* Cron -> Cloud Scheduler
* Image -> ...?

##### Users -> Identity-Aware Proxy?

上に書いたようにまったく同じことができるわけではない

##### Memcache -> Cloud Memorystore for Redis

ポジション的には正しいが、Cloud Memorystoreは現状、同じVPCからの接続しか許可していないので、App Engine Standardからは接続できない。
Cloud SQL Proxyのような機能が登場するのが待たれる。

##### Datastore -> Cloud Datastore

後ろ側は同じなので、機能的には同じですが、APIのInterfaceが異なるので、ソースコードは修正する必要があります。
[goon](https://github.com/mjibson/goon) , [nds](https://github.com/qedus/nds) を使っている場合は、 `google.golang.org/appengine` にがっつり依存しているので、Libraryたちが対応するまでは移行できません。
逆に考えるといつの日かLibraryが何かの対応をしてくれるかもしれないので、それまでのんびりしていてもよいかもしれません。

`google.golang.org/appengine` でも、 `cloud.google.com/go/datastore` でも差し替えれるようにした [go.mercari.io/datastore](https://github.com/mercari/datastore) を使っておくという手もあります。

##### Search -> ...?

形態素解析ベースの全文検索機能は現状GCPにはない。
公式ドキュメントでは自前でElastic Searchを運用するのがマイグレーション先になっている。

##### Mail -> SendGrid?

メール送信はずいぶん昔に非推奨になっているので、サードパーティに頼る形に。

##### TaskQueue -> Cloud Tasks

[Cloud Tasks](https://cloud.google.com/tasks/) がベータになっている。
ただ、まだPushしかない。
また、Cloud Tasksに移行すると、TaskQueue.AddがDatastore Transactionに参加できなくなる。

##### Cron -> Cloud Scheduler

[Cloud Scheduler](https://cloud.google.com/scheduler/) がベータになっている。

##### Image -> ...?

Image Serviceは地味に強力な機能で、ほぼ無料で動的にサムネイルなどを作れる強力な機能でした。
GCPの機能としては無いので、サードパーティの移行先として各種CDNのImage Optimizationを使うことを考える必要があります。

* [Akamai Image Manager](https://www.akamai.com/jp/ja/products/web-performance/image-manager.jsp)
* [Fastly Image Optimization](https://www.fastly.com/io)
* [SAKURA Internet Image Flux](https://www.sakura.ad.jp/services/imageflux/)

## 2nd Generationへのマイグレーション

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

## 2nd Generationが目指すゴール

2nd GenerationはGCPより前からあるApp Engineの資産を精算しようとしているように見えます。
App Engine Standard 1st Generationはそれだけで完成されたProductでしたが、GCPが登場してからの世界観とはなじまない部分が出てきています。
それを解決しようと [Managed VMs](https://qiita.com/sinmetal/items/68f0e21e1f33e3a553a1) そして [App Engine Flex](https://qiita.com/sinmetal/items/080a79702b060b33de69) が生まれたりもしましたが、完全には解決できませんでした。
2nd Generationは今のGCPと親和性の高いWeb ApplicationのためのPlatform as a Serviceを新たに作り出すのがゴールになりそうです。
ただ、その道は始まったばかりで、まだしばらくかかると思います。

## いつGo1.11にアップデートすべきか？

これから新しくApp Engineでアプリケーションを作り始める場合は、Go1.11で作り始めるのがおすすめです。

しかし、1st Generationで作られたアプリケーションを2nd Generationへの移行は大きなBreaking Changeを伴います。
まだ、1.11がベータで、1.9が使えなくなるのは2021年ぐらいだと思うので、そんなに急いでアップデートしなければいけないわけでもありません。
特に `login:admin` は移行先が難しいので、これに依存している場合は [Cloud Identity-Aware Proxy](https://cloud.google.com/iap/) が進化するのをしばらく待つのも一つの選択肢です。
`Go 1.11の機能がものすごく使いたい` `urlfetch, socketで困っている` `Cloud Spannerを使いたい` などの理由がない限りはもう少し待ってもいいかもしれません。

`login:admin を使っていない` `ログがばらばらになってもさほど気にならない` 場合は、割と簡単にマイグレーションが可能です。
ベータなのが気にならなければ、さくっとGo 1.11にしてしまってもよいでしょう。