package healthz

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type ServiceTarget struct {
	Name        string // "auth"
	Addr        string // "auth_service:8082"
	ServiceName string // "auth.AuthService"
}

type Checker struct {
	rdb     *redis.Client
	timeout time.Duration

	mu    sync.Mutex
	conns map[string]*grpc.ClientConn
}

func NewChecker(rdb *redis.Client, timeout time.Duration) *Checker {
	return &Checker{
		rdb:     rdb,
		timeout: timeout,
		conns:   make(map[string]*grpc.ClientConn),
	}
}

func (c *Checker) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cc := range c.conns {
		_ = cc.Close()
	}
	c.conns = make(map[string]*grpc.ClientConn)
	return nil
}

func (c *Checker) getConn(addr string) (*grpc.ClientConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cc, ok := c.conns[addr]; ok {
		return cc, nil
	}

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c.conns[addr] = cc
	return cc, nil
}

func (c *Checker) ReadyHandler(targets []ServiceTarget) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), c.timeout)
		defer cancel()

		resp := gin.H{
			"status":   "ok",
			"gateway":  "ok",
			"redis":    "unknown",
			"services": gin.H{},
		}

		// Redis readiness
		if c.rdb == nil {
			resp["redis"] = "fail: redis client is nil"
		} else if err := c.rdb.Ping(ctx).Err(); err != nil {
			resp["redis"] = "fail: " + err.Error()
		} else {
			resp["redis"] = "ok"
		}

		// Safe type assertion (errcheck-friendly)
		servicesAny, exists := resp["services"]
		if !exists {
			resp["status"] = "fail"
			ginCtx.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "fail",
				"error":  "internal: services field missing",
			})
			return
		}
		services, ok := servicesAny.(gin.H)
		if !ok {
			resp["status"] = "fail"
			ginCtx.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "fail",
				"error":  "internal: services field has unexpected type",
			})
			return
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, t := range targets {
			t := t
			wg.Add(1)
			go func() {
				defer wg.Done()

				cc, err := c.getConn(t.Addr)
				if err != nil {
					mu.Lock()
					services[t.Name] = "fail: " + err.Error()
					mu.Unlock()
					return
				}

				hc := grpc_health_v1.NewHealthClient(cc)
				r, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: t.ServiceName})
				if err != nil {
					mu.Lock()
					services[t.Name] = "fail: " + err.Error()
					mu.Unlock()
					return
				}

				mu.Lock()
				if r.Status == grpc_health_v1.HealthCheckResponse_SERVING {
					services[t.Name] = "ok"
				} else {
					services[t.Name] = "fail: " + r.Status.String()
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		// compute final status without races
		allOK := (resp["redis"] == "ok")
		for _, v := range services {
			if s, ok := v.(string); ok && s != "ok" {
				allOK = false
				break
			}
		}

		if allOK {
			ginCtx.JSON(http.StatusOK, resp)
			return
		}

		resp["status"] = "fail"
		ginCtx.JSON(http.StatusServiceUnavailable, resp)
	}
}
