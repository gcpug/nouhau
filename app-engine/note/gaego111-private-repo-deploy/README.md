# App Engine Standard Go 1.11 Private Repoを参照したDeploy

tag:["google-app-engine", "Go"]

App Engine Go Runtime 1.11から、Deployする時にGooble Cloud Buildを利用するようになっている。
`gcloud app deploy` を実行すると、暗黙的にCloud BuildにSubmitされる。
その時、go modulesを利用していると、go getが実行されるが、go getの対象にPrivate Repositoryがあると、getに失敗し `error loading module requirements` というメッセージが出る。
暗黙的に実行されるCloud BuildのJobには干渉できないので、Secret Tokenを渡したりすることもできない。
そのため、go getしないようにvendorを用意する必要がある。

### .gcloudignore

Cloud Build上でgo modulesが動かないように `.gcloudignore` に `go.mod` , `go,sum` を追加しておく 

```
go.mod
go.sum
```

`.gcloudignore` に `vendor` があると、Deployされないので、注意

### Deploy

vendorを作成し、 `GO111MODULE=off` を設定した状態で、 `gcloud app deploy` を行う

```
go mod vendor
GO111MODULE=off gcloud app deploy .
```

## Refs

* [go modulesを利用するとデプロイが遅くなる問題の解決策](https://github.com/gcpug/nouhau/issues/87)
