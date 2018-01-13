# nouhau
Google Cloud Platformのノウハウを共有するRepository
もし、あなたがGCPを初めて利用する場合、 https://github.com/gcpug/nouhau/tree/master/general/poem/link が役立つかもしれません。

## dir構成

以下のように置いてあるので、必要なノウハウを探してみてください。
欲しいノウハウが見つからなかった場合、issueに要望を書いておくと、暇な誰かが書いてくれたりすることがあります。

```
+ service // GCPのサービス名
  + middleware // gcsfuse, Ansibleなど特定のミドルウェアのノウハウの場合は1つ階層を掘ってまとめておく(option)
    + poem // 具体的な例示ではなく、読み物のようなノウハウ
      + title // コラムを端的に表したタイトル
        + README.md
        + ... // ソースコードや画像など
    + example // 実際に何かをやってみたexample
      + title // exampleを端的に表したタイトル
        + README.md
        + ... // ソースコードや画像など
```

## CONTRIBUTING

あなたがノウハウを持っている場合、Pull Reuqestを歓迎します。

https://github.com/gcpug/nouhau/blob/master/CONTRIBUTING.md

このVersionを使うとバグがあるといった、一時的な問題などは気軽にissueに書いておくだけでもかまいません。