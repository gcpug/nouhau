# Cloud BuildでEncrypted Valueの値を利用して、app.yamlを書き換える

tag:["google-cloud-build","google-cloud-kms"]

Google Cloud Buildを利用して、Build時にapp.yamlの内容を書き換えるサンプルです。
書き換える元の値はGoogle Cloud KMSで暗号化してcloudbuild.yamlに入れています。
書き換える箇所はApp Engine実行時に参照できる環境変数の値を入れる `env_variables` のValueです。
何かしらのSecret Tokenを入れてやるのに便利な方法です。
基本的には [公式ドキュメントのUsing Encrypted Secrets and Credentials](https://cloud.google.com/cloud-build/docs/securing-builds/use-encrypted-secrets-credentials) に書いてある通りの内容です。

また、 `secret.json` のようなファイルをビルド時に復号化する場合は [Encrypting a file using the CryptoKey](https://cloud.google.com/cloud-build/docs/securing-builds/use-encrypted-secrets-credentials?hl=en#encrypting_a_file_using_the_cryptokey) になります。

## config file example

app.yaml, cloudbuild.yamlは以下のとおりです。
`__SECRET_TOKEN__` をビルド時にsed commandを利用して置き換えます。

``` app.yaml
runtime: go111

env_variables:
  SECRET_TOKEN: "__SECRET_TOKEN__"

handlers:
- url: /.*
  secure: always
  script: auto
```

``` cloudbuild.yaml
steps:
  - name: 'ubuntu'
    entrypoint: 'bash'
    args: ['-c', 'sed -i -e s#__SECRET_TOKEN__#$$SECRET_TOKEN# app.yaml']
    secretEnv: ['SECRET_TOKEN']
  - name: gcr.io/cloud-builders/gcloud
    args: ['app', 'deploy', '--project=$PROJECT_ID', '--no-promote', '.']
secrets:
  - kmsKeyName: projects/gcpug/locations/global/keyRings/secret-token/cryptoKeys/password
    secretEnv:
      SECRET_TOKEN: CiQA7W/KKJF9AiNnV2upAQDJgWDcfxROYj7mvjvmPAr9OaMAKk4SMwAONeK/QabFca9zqpHPW2Q8dg/JofWARG4jaF5/vDGyxzyq9iW1JvVaTgLBwe2OAt8pDw==
```

## Secret Envの作り方

Secret Envは [Cloud KMS](https://cloud.google.com/kms/) を利用して作成します。
Cloud KMSについては [Cloud KMS Introduction](https://github.com/gcpug/nouhau/tree/master/cloudkms/poem/introduction) にも書いています。

### KeyRingとCryptoKeyの作成

KeyRingとCryptKeyは以下のコマンドで作成することができます。
この記事では以下の値で作成しています。

* [KEYRING-NAME] : secret-token
* [KEY-NAME] : password

```
gcloud kms keyrings create [KEYRING-NAME] \
  --location=global
```

```
gcloud kms keys create [KEY-NAME] \
  --location=global \
  --keyring=[KEYRING-NAME] \
  --purpose=encryption
```

### EntryptKeyにCloud Build Service AccountのRoleを追加

Cloud Build実行時にSecretEnvを復号化できるように、権限を追加します。

```
gcloud kms keys add-iam-policy-binding \
    password --location=global --keyring=secret-token \
    --member=serviceAccount:[SERVICE-ACCOUNT]@cloudbuild.gserviceaccount.com \
    --role=roles/cloudkms.cryptoKeyEncrypterDecrypter
```

### Secret Envの作成

いよいよSecret Envの作成です。
SecretTokenを暗号化する時に重要なのが、 `echo -n` です。
`-n` がないと末尾に改行が入ってしまいます。

```
echo -n "secrettoken" | gcloud kms encrypt \
  --plaintext-file=- \  # - reads from stdin
  --ciphertext-file=- \  # - writes to stdout
  --location=global \
  --keyring=secret-token \
  --key=password | base64
```

その結果、sed commandを実行すると以下のエラーが出て苦しめられます。

```
sed: -e expression #1, char 53: unterminated `s' command
```

そこさえ気をつければ、きっとできあがった文字列を `cloudbuild.yaml` の `secretEnv` に入れればOKです。