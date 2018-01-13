# Datastore Library Link

tag:["google-cloud-datastore"]

## Python

### NDB

https://cloud.google.com/appengine/docs/standard/python/ndb/

App Engine for PythonからDatastoreを利用する時の標準ライブラリ。
App Engine Memcacheを自動的に併用する機能が備わっている。

## Java

### Objectify

https://github.com/objectify/objectify

App Engine for JavaからDatastoreを利用する時に利用できるライブラリ。
NDBと同じような思想で作られているので、App Engine Memcacheを自動的に併用する機能が備わっている。

### Slim3

https://github.com/Slim3/slim3

App Engine for JavaからDatastoreを利用する時に利用できるライブラリ。
DatastoreだけでなくApp Engine API全般をラップしたフルスタックライブラリ。
日本人のエンジニアが中心となって作成しているので、日本語の記事が多い。
ただ、現在メンテナンスするメンバーが存在しない状態となってしまっている。

## Go

### goon

https://github.com/mjibson/goon

App Engine for GoからDatastoreを利用する時に利用できるライブラリ。
NDBと同じような思想で作られているので、App Engine Memcacheを自動的に併用する機能が備わっている。

### nds

https://github.com/qedus/nds

App Engine for GoからDatastoreを利用する時に利用できるライブラリ。
NDBと同じような思想で作られているので、App Engine Memcacheを自動的に併用する機能が備わっている。
goonよりインタフェースが [標準ライブラリ](https://godoc.org/google.golang.org/appengine/datastore) に近い。

### mercari/datastore

https://github.com/mercari/datastore

[標準ライブラリ](https://godoc.org/google.golang.org/appengine/datastore) , gonn, ndsに不満があった @vvakame さんの手によって作られたライブラリ。
App Engine for Goだけでなく、App Engine以外のGoからでも利用できるように設計されている。
ミドルウェアを差し込む機能が備わっており、App Engine Memcacheを併用する機能はミドルウェアとして差し込むことが可能になっている。
作者はGCPUG Slackにいるので、困ったら聞ける。

## PHP