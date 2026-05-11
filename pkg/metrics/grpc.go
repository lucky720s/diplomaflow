package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests handled by the server.",
		},
		[]string{"service", "method", "code"},
	)

	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_handling_seconds",
			Help:    "Histogram of gRPC request handling latency in seconds.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service", "method"},
	)

	grpcInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_server_in_flight_requests",
			Help: "Number of gRPC requests currently being processed.",
		},
		[]string{"service"},
	)
)

func GRPCCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		grpcRequestsTotal,
		grpcRequestDuration,
		grpcInFlight,
	}
}

func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		grpcInFlight.WithLabelValues(serviceName).Inc()
		defer grpcInFlight.WithLabelValues(serviceName).Dec()

		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		grpcRequestsTotal.WithLabelValues(serviceName, info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(serviceName, info.FullMethod).Observe(time.Since(start).Seconds())

		return resp, err
	}
}
