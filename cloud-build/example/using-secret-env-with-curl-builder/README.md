# gcr.io/cloud-builders/curl で secretEnvを利用する

tag:["google-cloud-build"]

Cloud BuildでCloud KMSを利用して環境変数を暗号化した状態で設定する機能がある。
設定方法は [Using Encrypted Resources](https://cloud.google.com/cloud-build/docs/securing-builds/use-encrypted-secrets-credentials) に書かれており、Secretな値をBuild時に利用したい場合、とても便利な機能だ。

ただ、 `gcr.io/cloud-builders/curl` でsecretEnvを利用する時にハックが必要になる。
ユースケースとしては、Basic認証がかかっているURLにcurlを投げる時などはPASSWORDの値をsecretEnvに保存したくなるが、単純にsecretEnvを設定して取得しようとしても、取得できない。

## 解決方法

`entrypoint: bash` にして、 `curl` を実行する。

### cloudbuild.yaml

```
steps:
- name: gcr.io/cloud-builders/curl
  entrypoint: bash
  args: ['-c', 'curl --silent -u user:$$PASSWORD https://gcpug.jp']
  secretEnv: ['PASSWORD']
secrets:
- kmsKeyName: projects/sinmetal-kms/locations/global/keyRings/sample-key-ring/cryptoKeys/sample-key
  secretEnv:
    PASSWORD: CiQA7W/KKF9f37KCiwJLVkZLisM+Cgzm0sAZ4wKXAllbuYZPzJwSLgBGfWCMklKOyHJz5qvQMV13uQ0fxC7TWFefXPcpwQbrsPrKG00COwQudB9uSrA=
```

Refs https://github.com/GoogleCloudPlatform/cloud-builders/issues/312