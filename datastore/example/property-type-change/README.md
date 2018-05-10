# Property Type Change

tag:["google-cloud-datastore", "Go"]

Datastoreに保存しているPropertyの型を途中で変更した場合に、Entithy取得時に変換を行うサンプル

## Go

intで保存していたPropertyをfloat64に変更するサンプル。
変換していない場合、実行時に `datastore: cannot load field "Value" into a "datastore.HogeV2": type mismatch: int versus float64` というエラーが発生する。
解決策としては、structに `datastore.PropertyLoadSaver` を実装すれば、structとEntityの変換処理をカスタマイズできるので、Load時に型がintの場合、float64に変換するようにしている。

なお、DatastoreではEntityのPropertyはint, int32などでも一律int64として保存されている。
float32についても同様にfloat64として保存されている。
このため、変換処理についてはたとえintで保存していてもint64だけ面倒を見ればよい。
[Propertyのドキュメント](https://godoc.org/cloud.google.com/go/datastore#Property)を参照のこと。

```
type HogeV1 struct {
	Value int
}

type HogeV2 struct {
	Value float64
}

var _ datastore.PropertyLoadSaver = &HogeV2{}

func (h *HogeV2) Load(ps []datastore.Property) error {
	for _, v := range ps {
		if v.Name == "Value" {
			switch i := v.Value.(type) {
			case int64:
				v.Value = float64(i)
			default:
				// noop
			}
		}
	}

	return datastore.LoadStruct(h, ps)
}

func (h *HogeV2) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(h)
}

func TestConvertPropertyType(t *testing.T) {
	ctx := context.Background()

	ds, err := datastore.NewClient(ctx, "testproject")
	if err != nil {
		t.Fatal(err)
	}

	key := datastore.NameKey("Hoge", "key1", nil)
	_, err = ds.Put(ctx, key, &HogeV1{
		Value: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	var h HogeV2
	if err := ds.Get(ctx, key, &h); err != nil {
		t.Fatal(err)
	}
	if e, g := 10.0, h.Value; e != g {
		t.Fatalf("expected Value is %f, got %f", e, g)
	}
}
```