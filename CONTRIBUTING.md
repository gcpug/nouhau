# Contributing to nouhau

nouhauへのContributingに興味を持っていただきありがとうございます！
私たちはあなたのpull requestを歓迎します。

## dir構成

* ノウハウを探しやすいようにディレクトリで分類しておく。
* 複数のGCPサービスを利用したノウハウについては、いったん一番こいつが主体かなーと思うものを選択しておく。 (後でcompositeみたいなdirを作って、その中にまとめるかも)
* ノウハウの内容はディレクトリの中にREADME.mdを置いて、そこに書いていく。README.mdのテンプレートとして [README-TEMPLATE.md](README-TEMPLATE.md) を利用する。

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

### Example

Google Cloud Storageのgcsfuseを使う時に最初に気をつけるべきことというノウハウを共有する時

```
+ storage
  + gcsfuse
    + poem
      + first-step
        + README.md
```

BigQueryでテキストにURLエンコードされた文字列が入っているから、そいつとなんとかする時の方法についてのノウハウを共有する時

```
+ bigquery
  + example
      + url-decord
        + README.md
        + url-decord.sql
```

## Pull Request ガイドライン

### 新しいノウハウを作成する

あなたが新しいノウハウを登録したいと思った時に、新しいpull requestを作成します。
issueは必ずしも立てる必要はありませんし、brach名も特にしばりはありません。
dir構成に従って新しいdirを作成し、その中にノウハウを書き込みpull requestを送れば、OKです。
pull requestはGCPUG Memberによってレビューが行われた後、Mergeされます。

### 既存のノウハウを修正する

typoを発見した時や、コマンドが古くなっているのを見つけた時、また、記事が間違っていることに気付いた時などに既存のノウハウを修正するpull requestを投げます。
pull requestは記事を書いた人がレビュアーとしてアサインされ、レビューされた後、Mergeされます。