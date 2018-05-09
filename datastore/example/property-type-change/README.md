# Property Type Change

tag:["google-cloud-datastore", "Go"]

Datastoreに保存しているPropertyの型を途中で変更した場合に、Entithy取得時に変換を行うサンプル

## Go

intで保存していたPropertyをfloat64に変更するサンプル。
変換していない場合、実行時に `datastore: cannot load field "Value" into a "datastore.HogeV2": type mismatch: int versus float64` というエラーが発生する。
解決策としては、structに `datastore.PropertyLoadSaver` を実装すれば、structとEntityの変換処理をカスタマイズできるので、Load時に型がintの場合、float64に変換するようにしている。

```
package datastore

import (
	"context"
	"testing"

	"cloud.google.com/go/datastore"
)

type HogeV1 struct {
	Value int
}

type HogeV2 struct {
	Value float64
}

var _ datastore.PropertyLoadSaver = &HogeV2{}

func (h *HogeV2) Load(ps []datastore.Property) error {
	var nps []datastore.Property
	for _, v := range ps {
		if v.Name == "Value" {
			switch i := v.Value.(type) {
			case int:
				v.Value = float64(i)
			case int8:
				v.Value = float64(i)
			case int16:
				v.Value = float64(i)
			case int32:
				v.Value = float64(i)
			case int64:
				v.Value = float64(i)
			default:
				// noop
			}
		}
		nps = append(nps, v)
	}

	err := datastore.LoadStruct(h, nps)
	if err != nil {
		return err
	}

	return nil
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
		t.Fatalf("expected Value is %d, got %d", e, g)
	}
}
```