package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total HTTP requests processed by api_gateway",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	gatewayRouteInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_route_info",
			Help: "Static list of api_gateway routes (1 = exists). Used to show 0-values in dashboards.",
		},
		[]string{"method", "path"},
	)
)

func MustRegisterGatewayMetrics(reg prometheus.Registerer) {
	reg.MustRegister(httpRequestsTotal, httpRequestDuration, gatewayRouteInfo)
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
	}
}

func SetRouteExists(method, path string) {
	gatewayRouteInfo.WithLabelValues(method, path).Set(1)
}
