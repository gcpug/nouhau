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
package datastore

import (
	"context"
	"testing"

	"cloud.google.com/go/datastore"
)

// HogeV1 is Old Entity struct
type HogeV1 struct {
	Value int
}

// HogeV2 is New Entity struct
type HogeV2 struct {
	Value float64
}

var _ datastore.PropertyLoadSaver = &HogeV2{}

// Load is datastore.PropertyLoadSaver を満たすためのfunc
// Entityをstructに変換する処理
func (h *HogeV2) Load(ps []datastore.Property) error {
	for idx, v := range ps {
		if v.Name == "Value" {
			switch i := v.Value.(type) {
			case int64:
				v.Value = float64(i)
			default:
				// noop
			}
			ps[idx] = v
		}
	}

	return datastore.LoadStruct(h, ps)
}

// Save is datastore.PropertyLoadSaver を満たすためのfunc
// structをEntityに変換する処理
func (h *HogeV2) Save() ([]datastore.Property, error) {
	return datastore.SaveStruct(h)
}

// TestConvertPropertyType is Sample Test
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