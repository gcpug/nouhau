# App Engine Standard Go 1.11 Private Repoを参照したDeploy

tag:["google-app-engine", "Go"]

App Engine Go Runtime 1.11から、Deployする時にGooble Cloud Buildを利用するようになっている。
`gcloud app deploy` を実行すると、暗黙的にCloud BuildにSubmitされる。
その時、go modulesを利用していると、go getが実行されるが、go getの対象にPrivate Repositoryがあると、getに失敗し `error loading module requirements` というメッセージが出る。
暗黙的に実行されるCloud BuildのJobには干渉できないので、Secret Tokenを渡したりすることもできない。
そのため、go getしないようにvendorを用意する必要がある。

以下のような形で、vendorを作成し、 `go.mod` `go.sum` を削除しておけば、Cloud Buildの中でgo getが走らない。

```
go mod vendor
rm go.mod go.sum
```

`go.mod` `go.sum` に関しては `app.yaml` と同じ階層になければ、消さなくてもよいという情報もあるが、細かいディレクトリ構成による差異については未検証。

## Refs

* [go modulesを利用するとデプロイが遅くなる問題の解決策](https://github.com/gcpug/nouhau/issues/87)
