# GCP Tips

tag["google-cloud-platform"]

## Google Cloud ConsoleはProject Numberでも開ける

https://console.cloud.google.com/home/dashboard?project=$PROJECT_NUMBER

のような感じでProjectNumberを指定しても開ける

## Google Cloud SDKのプロパティは環境変数で設定できる

https://cloud.google.com/sdk/docs/properties#setting_properties_via_environment_variables

例えば以下のようなものが指定できる

```
CLOUDSDK_CORE_PROJECT=gcpug
CLOUDSDK_COMPUTE_ZONE=asia-northeast1-a
CLOUDSDK_SPANNER_INSTANCE=public-spanner
CLOUDSDK_AUTH_IMPERSONATE_SERVICE_ACCOUNT=client@gcpug.iam.gserviceaccount.com
CLOUDSDK_BILLING_QUOTA_PROJECT=gcpug
```
