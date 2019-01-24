# Cloud Functions でパッケージ変数をエントリポイントに指定する
Cloud Functions を Go で動かす場合に指定するエントリポイントは以下の条件を満たす必要があります。

1. 型を満たしている
    * httpリクエストを受ける場合は `func(http.ResponseWriter, *http.Request)`型
1. パッケージ外に公開されているものであること
    * 関数をmainパッケージではないパッケージに作らないといけないので、恐らくそうでしょう。

なので、この条件を満たしていればパッケージ変数でもエントリポイントに指定することができます。

検証コード
```
package functions

import (
	"fmt"
	"net/http"
)

var F = func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello world")
}
```
このコードを `--entry-point=F` を指定してhttpをトリガーとしてデプロイします。
デプロイした関数にアクセスするときちんと「hello world」を返してくれます。

init関数でパッケージ変数に代入することで、デプロイ時に1度だけ初期化するエントリポイントを作ることができます。
