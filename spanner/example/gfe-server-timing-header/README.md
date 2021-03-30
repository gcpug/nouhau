# GFE Server-Timing Header

tag:["google-cloud-spanner"]

Spanner API利用時にGoogle Frontend Serverが返してくるLatency Metricsの値をgRPCのHeaderから取得するサンプル。
元は [Use GFE Server-Timing Header in Cloud Spanner Debugging](https://medium.com/google-cloud/use-gfe-server-timing-header-in-cloud-spanner-debugging-d7d891a50642) から来ている。
記事内のGoのサンプルは [cloud.google.com/go/spanner/apiv1](https://pkg.go.dev/cloud.google.com/go/spanner/apiv1) を利用しているが、普段よく使うのは [cloud.google.com/go/spanner](https://pkg.go.dev/cloud.google.com/go/spanner) だと思うので、 `cloud.google.com/go/spanner` で取得できるサンプルを書いた。

`cloud.google.com/go/spanner` ではgRPCのHeaderを取得するインターフェースは露出していないので、gRPC Interceptorを利用して取得する。
今回のサンプルでは取得した値を [OpenCensus](https://opencensus.io/) を利用して、TraceのAttributeに追加している。

``` Go
package main

import (
	"context"
	"log"
	"strconv"
	"strings"

	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GFEMetricsUnaryClientInterceptor is server-timing をログ出力する
// https://medium.com/google-cloud/use-gfe-server-timing-header-in-cloud-spanner-debugging-d7d891a50642
func GFEMetricsUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var md metadata.MD
		addOpts := append(opts, grpc.Header(&md))

		defer func() {
			v, ok := ExtractServerTimingValue(md)
			if !ok {
				return
			}
			span := trace.FromContext(ctx)
			span.AddAttributes(trace.Int64Attribute("server-timing", v))
		}()

		return invoker(ctx, method, req, reply, cc, addOpts...)
	}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	md, err := w.Header()
	if err != nil {
		log.Printf("failed grpc.ClientStream.Header.Get")
		return w.ClientStream.RecvMsg(m)
	}
	v, ok := ExtractServerTimingValue(md)
	if ok {
		span := trace.FromContext(w.Context())
		span.AddAttributes(trace.Int64Attribute("server-timing", v))
	}
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	return w.ClientStream.SendMsg(m)
}

// wrappedStream  wraps around the embedded grpc.ClientStream, and intercepts the RecvMsg and
// SendMsg method call.
type wrappedStream struct {
	grpc.ClientStream
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}

// GFEMetricsStreamClientInterceptor is server-timing をログ出力する
// https://medium.com/google-cloud/use-gfe-server-timing-header-in-cloud-spanner-debugging-d7d891a50642
func GFEMetricsStreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		s, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		return newWrappedStream(s), nil
	}
}

func ExtractServerTimingValue(md metadata.MD) (int64, bool) {
	metaValues := md.Get("server-timing")
	for _, mv := range metaValues {
		v := strings.ReplaceAll(mv, "gfet4t7; dur=", "")
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	}
	return 0, false
}
```