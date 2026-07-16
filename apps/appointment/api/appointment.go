// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/handler"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/middleware"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/appointment-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// ================= 微服务治理中间件注册 =================

	// 1. 灰度路由中间件 — 10% 流量染色到 Canary 实例
	canaryMw := middleware.NewCanaryMiddleware()
	server.Use(canaryMw.Handle)

	// 2. 限流中间件 — IP + 用户双层限流（反黄牛）
	rateLimitMw := middleware.NewRateLimitMiddleware(
		c.BizRedis,
		c.RateLimit.Period,
		c.RateLimit.IpQuota,
		c.RateLimit.UserQuota,
	)
	server.Use(rateLimitMw.Handle)

	// 3. 熔断器中间件 — 监控下游错误率，超阈值自动熔断
	breakerMw := middleware.NewBreakerMiddleware(
		c.GatewayBreaker.Threshold,
		c.GatewayBreaker.Cooldown,
	)
	server.Use(breakerMw.Handle)

	// ================= 业务路由注册 =================
	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// ================= 管理端点注册 =================
	adminMux := http.NewServeMux()
	handler.RegisterAdminRoutes(adminMux, ctx, breakerMw)
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/v1/admin/status",
			Handler: adminMux.ServeHTTP,
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/admin/simulate/failure",
			Handler: adminMux.ServeHTTP,
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/admin/simulate/recovery",
			Handler: adminMux.ServeHTTP,
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/admin/breaker/reset",
			Handler: adminMux.ServeHTTP,
		},
	})

	fmt.Printf("🛡️  微服务治理组件已加载:\n")
	fmt.Printf("    ├── 🔍 服务注册发现: etcd (2379)\n")
	fmt.Printf("    ├── 🚦 IP限流: %d次/%ds | 用户限流: %d次/%ds\n", c.RateLimit.IpQuota, c.RateLimit.Period, c.RateLimit.UserQuota, c.RateLimit.Period)
	fmt.Printf("    ├── 🛡️  网关熔断: threshold=%d, cooldown=%s\n", c.GatewayBreaker.Threshold, c.GatewayBreaker.Cooldown)
	fmt.Printf("    ├── 🔀 灰度发布: 10%% 流量 → Canary 实例\n")
	fmt.Printf("    └── 📊 管理端点: http://localhost:%d/v1/admin/status\n\n", c.Port)

	fmt.Printf("Starting API Gateway at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
