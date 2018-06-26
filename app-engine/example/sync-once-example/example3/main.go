package myapp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tenntenn/sync/recoverable"
	"github.com/tenntenn/sync/try"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

func init() {
	http.HandleFunc("/_ah/warmup", warmup)
	http.HandleFunc("/", index)
}

var (
	msg  string
	once try.Once
)

func getMsg(ctx context.Context) {
	panic("panic")
	/*
		_, err := memcache.JSON.Get(ctx, "msg", &msg)
		if err != nil {
			log.Errorf(ctx, "Error: %v", err)
			panic(err)
		}
		log.Infof(ctx, "getMsg: msg is %s", msg)
	*/
}

func warmup(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	if err := once.Try(recoverable.Func(func() { getMsg(ctx) })); err != nil {
		log.Infof(ctx, "Error: %v", err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	var _msg string
	if err := once.Try(recoverable.Func(func() { getMsg(ctx) })); err != nil {
		log.Infof(ctx, "Error: %v", err)
		_msg = "default value"
	} else {
		_msg = msg
	}
	log.Infof(ctx, "msg: %s", _msg)
	fmt.Fprintln(w, "msg:", _msg)
}
