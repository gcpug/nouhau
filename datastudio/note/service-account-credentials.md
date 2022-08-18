# DataStudio Service Account credentials

2022年6月に [Google Cloud service account data credentials](https://support.google.com/datastudio/answer/11521624#jun-20-2022&zippy=%2Csee-older-releases) として、 BigQuery データソースのジョブを発行するサービスアカウントを設定できるようになった。

長い間の BigQuery データソースはオーナーのユーザアカウントでジョブを発行する Owner credentials とレポート閲覧者のユーザアカウントでジョブを発行する Viewer credentials のどちらかで動作しており、サービスアカウントを使う設定は存在しなかったため、待望の設定といえる。

[ドキュメント](https://support.google.com/datastudio/answer/10835295?hl=ja)では公式にはこの3点を代表的メリットとして上げている。
> * サービス アカウントの認証情報を使用しているデータソースは、作成者が退職した場合でも悪影響を受けることはありません。
> * サービス アカウントの認証情報を使用すると、デバイス ポリシーを使用する VPC Service Controls の境界の背後にあるデータにアクセスできます。
> * スケジュール設定されたメールやスケジュール設定されたデータ抽出などの自動機能では、VPC Service Controls の境界の背後にあるデータソースが使用されます。

他にもユーザアカウントの BigQuery ジョブの Personal History (`bq ls -j` 相当)が DataStudio が自動的に発行するジョブに汚染されないというメリットがあると考えられる。

## 設定方法

[データの認証情報](https://support.google.com/datastudio/answer/6371135?hl=ja) のドキュメントには

> サービス アカウントは、Google Workspace または Cloud Identity の管理者が設定する必要があります。その場合は、サービス アカウントの設定方法をご確認ください。

と書いてあるが、実際には Google Workspace 側の管理者権限は必要がない。おそらく GCP に不慣れな DataStudio ユーザ向けは管理者に設定してもらうように、という意味の記述ではないかと思われる。

実際には[データポータル用に Google Cloud サービス アカウントを設定する](https://support.google.com/datastudio/answer/10835295) のドキュメントにあるように、

* DataStudio Service Agent サービスアカウントの email を確認する。
  * Workspace または Cloud Identity のユーザーであれば [データポータル サービス エージェントのヘルプページ](https://datastudio.google.com/c/serviceAgentHelp) にアクセスすれば取得できる。
  * 通常 `service-org-${ORG_NUMBER}@gcp-sa-datastudio.iam.gserviceaccount.com` の形式となる。
* DataStudio Service Agent サービスアカウントに対象のサービスアカウントの Service Account Token Creator ロールを与える。
* DataStudio 上で Service Account credentials を設定したいユーザアカウントに対象のサービスアカウントの Service Account User ロールを与える。
* 対象のサービスアカウントには必要な BigQuery ロール（例: BigQuery Data Viewer, BigQuery Job User）を与える。

のように、通常の GCP のサービスアカウント周りの権限設定の範疇で全て設定できる。
