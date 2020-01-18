# GAE 1st Gen Migration Plan Discussion

https://gcpug-tokyo.connpass.com/event/157556/ で話し合った内容のメモ

Slide : https://docs.google.com/presentation/d/1c-l50-w6TmXnKNk10YTiPlLZxjokYZXQrwc8YFtB2mw/edit#slide=id.g6d836b302d_0_188

## 大事なこと

2020年1月時点では2nd genに急いでmigrationする必要はない。
しかし、情報を集めプランを練っていく必要はある。

GCPの1st genがシャットダウンされる日が来るのはかなり先だろう。
使っているLibraryなどが1st genのRuntimeのVersionをサポートしなくなることの方を気にした方がよい。

## 参加者のRuntime内訳 (多い順)

* Go
* Python
* Java
* PHP (Zero)

## ディスカッションメモ

### login:admin

#### IAP(Identity Aware Proxy)を利用するケース

ServiceごとにIAMを変更できるので、admin用のServiceを作る。
PublicにしたいやつはallUsersを設定してやる。
Cloud Tasks, Cloud PubSubから受け取る方法なんかは [GCP からの HTTP リクエストをセキュアに認証する](https://medium.com/google-cloud-jp/gcp-%E3%81%8B%E3%82%89%E3%81%AE-http-%E3%83%AA%E3%82%AF%E3%82%A8%E3%82%B9%E3%83%88%E3%82%92%E3%82%BB%E3%82%AD%E3%83%A5%E3%82%A2%E3%81%AB%E8%AA%8D%E8%A8%BC%E3%81%99%E3%82%8B-dda4933afcd6) にまとめてくれてたりする。
/admin みたいにpathで振り分けたい場合はdispatch.yamlでServiceを振り分けてあげる。

BlobstoreがうまいことAllUsersを抜けてくれないという報告も上がっていた。

#### Cloud Run, Cloud Functionも併用

元々、App EngineのServiceを分割してる場合は、Cloud TasksやPubSubを受け取るものは、個別にRunやFunctionにしてしまう戦略

#### 元々、自分で認証書いてた

App Engine Flexはずいぶん昔にlogin:adminが廃止されているので、自分で認証系の実装している

### App Engine Search API

#### Elastic Cloud

料金の支払いはGCPに統合できるようになっている。
ただ、接続がPublicにIDとパスワードを送る方式なのが気になるところ。
Google Compute Engineで動いているのだろうから、Googleのネットワークから出てないとは言え、VPC ピアリングとかして欲しい。

#### Elasticsearch on Google Compute Engine

自分でElasticsearchなどのミドルウェアを動かすという手。

#### [Algolia](https://www.algolia.com/)

Firestoreのドキュメントにもレコメンドとして登場するFull Text SearchのSaaS
とりあえず、さくっと検索作りたい時は便利。
Tokyo Regionも存在する。
日本語の検索が若干弱かったり、料金体系ががらっと変わったりとか、なんかうまくいかんな？という時にデータを再度全投入したりする必要があったみたいな声も。

#### 己で作る

Datastore, SpannerなどのDatabaseに自分で全文検索INDEX作って、検索する人々。

### Task Queue with Datastore Tx

[Features in Task Queues not yet available via Cloud Tasks API](https://cloud.google.com/tasks/docs/migrating?hl=en#features_in_task_queues_not_yet_available_via)

#### おらおらお実装する

Taskの内容をDatastoreに保存しておき、TaskのHandlerはそこを見て、実行するかどうかを判断するみたいな実装をする。
大抵、先にTaskをAddして、後でDatastoreのTxをかけることになると思うが、Task Handler側をDatastoreのTxが成功したであろう時間の後で実行してやる必要があったりするので、調整は必要。

[Cloud Tasksのタスク追加をCloud Datastoreトランザクションに含める](https://qiita.com/hogedigo/items/2202cc62dc19d90e421d) とかやってる人もいた。

#### log -> Stackdriver Logging Sink -> PubSub

log出力は失敗しないだろう！Stackdriver Logging Sinkは確実にPubSubに入れてくれるだろう！という信頼のもとに実装されたダーティなアーキテクチャ。
とりあえずは、問題なく動いているように見えているが、そもそも問題が起こってるかどうかが分かりにくいという問題もある

### Performance

2nd gen(gVisor版)はPerformanceが悪くなる部分があるというのが、ちょいちょい話題に上がっている。
CPU利用率が高くなったり、メモリ使用量が増えたりする。(その影響かは分からないが、App Engine Standardは料金そのままでメモリが倍になっている)
gVisorのせいだとするとネットワークとかが遅くなるような気がするけど、Go Runtimeで正規表現とかrefrectとか使っているJson Validationがめっちゃ遅くなったりもした。
cryptoとかはなぜか早くなった。

2nd genにするとGoogle Cloud Client LibraryとかgRPCとかのレイヤーの可能性もあるので、いまいち原因を突き止めるのが難しい。
ひとまずは、Stackdriver ProfilerとかTraceとかでボトルネックとかを探すみたいなことをやっていくしかない。

### Observability

App Engine 1st genのLogに憧れて、みんな頑張ってる。
[GAE のログに憧れて](https://medium.com/google-cloud-jp/gae-%E3%81%AE%E3%83%AD%E3%82%B0%E3%81%AB%E6%86%A7%E3%82%8C%E3%81%A6-895ebe927c4)

* Python : https://github.com/gumo-py/gumo-logging
* Go : https://github.com/yfuruyama/crzerolog
* Go : https://github.com/DeNA/aelog
* Go : https://github.com/vvakame/sdlog
* Go : https://github.com/aikizoku/rabbitgo/blob/master/appengine/default/src/lib/log/writer_stackdriver.go

### urlfetch

疎まれていたことが多いurlfetch, 実はApp Engine Service間で通信する時に、認証してくれてた。

urlfetch無しでも、Metadata ServerからID Tokenを取ってHeaderに付けて〜みたいなことすればOK。
ただ、App Engine全体で同じものだから、Serviceごとに権限を変えたい場合は困る。

### memcache

現状、GCPでKey ValueのExpireする高速DBとしては [Cloud Memorystore for Redis](https://cloud.google.com/memorystore/docs/redis/redis-overview?hl=en) になる。
Memosytoreは同じVPCからしかアクセスので、App Engine Standardの場合は [Serverless VPC Access](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access?hl=en) を作って、VPCに接続してやる。
app.yamlの設定としては [Configuring your app to use a connector](https://cloud.google.com/appengine/docs/standard/go/connecting-vpc?hl=en#configuring) に書いてある。

ただ、App Engine Memcacheと違って、Free Quotaは存在せず、Nodeのサイズみたいな概念もある(VMにRedisが乗っかってる感じのサービス)ものなので、そこを管理してあげる必要はある。
Memorystoreはあまりアップデートがないのもちょっと不安な存在である。

### Image Service, Blobstore Service

Image Serviceは企業勢はCDNたちが同じようなサービスを出しているので、そっちに移行するのが無難。
個人勢はメモリ上で適当に変換したりして、エッジキャッシュとかに乗せるみたいなノリが無難だろうか。

Blobstoreは [Cloud Storage Signed URL](https://cloud.google.com/storage/docs/access-control/signed-urls?hl=en) で同じようなノリのことをすることになりそう。

### Local Dev

App Engine SDKというものは消滅し、各種Runtimeの標準的な方法を利用するようになっている。
ただ、app.yamlにstatic serverの設定書いてる部分とかは反映されないので、なんか適当にやってやる必要がある。

GoだとLocal用とServer用でmain関数を分けて、完全に動きを分けたりしてる。
Local起動の場合は、Goでstatic contentsを返してあげる。

### Compute

Cloud Run, Firebaseに興味を持っている人は多い。
ハイブリッドを狙っていく感じ。
GKEへの移行という手もあるけど、GAEのゆりかごに慣れた人が、そのままの気持ちでGKEに行くと、自分で舵を取る必要があるので、かなり大変だろう。

### Database

大半のメンバーはDatastoreを使い続ける予定。
一部、Spannerに移行する人もいる感じ。

Datastoreの各種ライブラリ利用者の現状のアプローチは以下のような雰囲気。

#### slim3 (Java)

現状、ノープラン

#### Objectify (Java)

会場にはたぶん使ってる人いなかった

#### ndb (Python)

公式での対応待ち

#### goon (Go)

メインコミッターの人は更新する予定はないはず。
goonはやめて、Cloud DatastoreのClient Libraryに移行する予定で、cacheレイヤーをgRPCのインターセプターに差し込んでやろうと思って、 https://github.com/DeNA/cloud-datastore-interceptor を作ったりしている。

#### nds (Go)

会場ではたぶん使ってる人いなかった

#### mercari/datastore (Go)

Clientの持ち回り方法とかを変えるだけでいける。
