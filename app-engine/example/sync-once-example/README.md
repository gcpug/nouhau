# インスタンスごとの初期化処理

tag:["google-app-engine"]

See: [sync.Onceとエラー処理](https://docs.google.com/presentation/d/1IXumqwmILh7Lxisn2MMun4n4CQVOa4pV9JAPUYSxL_0/edit?usp=sharing)

## sync.Onceを使う

see: [example1](./example1)

インスタンス起動時にくるウォームアップリクエストで初期化を行う。
ウォームアップリクエストを受け取るにはapp.yamlに以下を設定する。

```yaml
inbound_services:
- warmup
```

`/_ah/warmup`にHTTPハンドラをしかける。

```go
http.HandleFunc("/_ah/warmup", warmup)
```

## sync.Onceでの問題点

see: [example2](./example2)

* panicが発生した場合に検出できない
 * 自前でラップするしかない
 * panicはゴールーチンごとにしか[recoverできない](https://play.golang.org/p/SuFarBNsSAv)
* エラー処理ができない
 * 戻り値としてエラーが返せない

## tenntenn/sync/recoverableとtenntenn/sync/tryを使う

see: [example3](./example3)

### tenntenn/sync/recoverable

* panicをエラーに変換するライブラリ
 * https://godoc.org/github.com/tenntenn/sync/recoverable

```go
func main() {
	// func() -> func() error
	f := recoverable.Func(func() { panic("example") })
	if err := f(); err != nil {
		if v, ok := recoverable.RecoveredValue(err); ok {
			fmt.Println("panic with ", v)
		}
	}
}
```

### tenntenn/sync/try

* エラー発生時に検出できるOnceを提供する
 * https://godoc.org/github.com/tenntenn/sync/try

```go
func main() {
	var once Once
	for i := 1; i <= 3; i++ {
		i := i
		err := once.Try(func() error {
			if i < 3 { return errors.New("error") }
			return nil
		})
		if err != nil { fmt.Printf("try %d %v\n", i, err) }
	}
}
```
