# Ingress の(バッド)ノウハウ

GKE のドキュメントには Ingress を使って GCE HTTP(S) LB を使って外部にサービスを公開する方法が書かれている。

- https://cloud.google.com/kubernetes-engine/docs/tutorials/http-balancer?hl=en
- https://cloud.google.com/kubernetes-engine/docs/tutorials/configuring-domain-name-static-ip?hl=en

しかし、ヘルスチェック等の落とし穴にハマる人が多く、 GKE に予め入っている Ingress の実装であるingress-gce の README.md を読みに行かないと分からないことが多い。

https://github.com/kubernetes/ingress-gce/blob/master/README.md

## 主な注意事項
### GCE の HTTP(S) Load Balancing の様子を見るとヘルスチェックが失敗している
- GCE Ingress のヘルスチェックの挙動を確認すること https://github.com/kubernetes/ingress-gce/blob/master/README.md#health-checks
- ヘルスチェックのパスはデフォルトで `/`
  - `readinessProbe` の `httpGet` を設定した場合は変更可能
- ヘルスチェックのステータスコードは 200 でないと失敗する
  - GCE ヘルスチェックの仕様
    - https://cloud.google.com/compute/docs/load-balancing/health-checks?hl=en#http_and_https_health_checks
  - `readinessProbe` や他のクラウドプロバイダでは 2xx, 3xx なら成功なので引っかかりがち
### 更新が反映されずに作り直す必要がある場合がある

- `kubectl apply -f` や `kubectl edit` での更新が反映されない時は `kubectl delete` & `kubectl create` もしくは `kubectl replace --force` で作り直す(IP アドレスに注意)

### Ingress を削除する前に GKE クラスタを削除すると色々と GCE のリソースが残る。

- Ingress が完全に削除されたことを確認してから GKE クラスタを削除する
- リソースが残ってしまった時のために公式にスクリプトが用意されそう
  - https://github.com/kubernetes/kubernetes/pull/62871

### ソースの IP アドレスが正しく取れない
- GCE HTTPS LB および Kubernetes 内の NAT によってソース IP アドレスが変わってしまう
  - `x-forwarded-for` ヘッダの値を使う https://cloud.google.com/compute/docs/load-balancing/http/?hl=en#components
    - HTTP(S) LB の Proxy の IP アドレスなので `130.211.0.0/22`, `35.191.0.0/16` は nginx real ip module 等で飛ばして良い
      - https://cloud.google.com/compute/docs/load-balancing/http/?hl=en#troubleshooting
  - Service に `service.spec.externalTrafficPolicy: Local` を設定する 
    - https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip
    - https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-typenodeport 
- 原理は The Ins and Outs of Networking in Google Container Engine ([YouTube](https://www.youtube.com/watch?v=y2bhV81MfKQ), [SpeakerDeck](https://speakerdeck.com/thockin/the-ins-and-outs-of-networking-in-google-container-engine)) 参照
