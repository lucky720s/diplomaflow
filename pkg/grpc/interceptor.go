package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func DefaultRetryServiceConfig() string {
	return `{
		"methodConfig": [{
			"name": [{"service": ""}],
			"retryPolicy": {
				"MaxAttempts": 4,
				"InitialBackoff": "0.1s",
				"MaxBackoff": "2s",
				"BackoffMultiplier": 2.0,
				"RetryableStatusCodes": ["UNAVAILABLE", "RESOURCE_EXHAUSTED", "ABORTED"]
			}
		}]
	}`
}

func TimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
