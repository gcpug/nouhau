package myapp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

func init() {
	http.HandleFunc("/_ah/warmup", warmup)
	http.HandleFunc("/", index)
}

var (
	msg  string
	once sync.Once
)

func getMsg(ctx context.Context) {
	_, err := memcache.JSON.Get(ctx, "msg", &msg)
	if err != nil {
		log.Errorf(ctx, "Error: %v", err)
		panic(err)
	}
	log.Infof(ctx, "getMsg: msg is %s", msg)
}

func warmup(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	once.Do(func() { getMsg(ctx) })
}

func index(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	once.Do(func() { getMsg(ctx) })
	log.Infof(ctx, "msg: %s", msg)
	fmt.Fprintln(w, "msg:", msg)
}
