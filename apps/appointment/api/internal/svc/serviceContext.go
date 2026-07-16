// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"

	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/appointment"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config           config.Config
	RedisClient      *redis.Redis          // Redis 客户端（供限流中间件等使用）
	CanaryMiddleware rest.Middleware
	StableRpc        appointment.Appointment
	CanaryRpc        appointment.Appointment
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		RedisClient: redis.MustNewRedis(c.BizRedis),
		StableRpc: appointment.NewAppointment(zrpc.MustNewClient(c.AppointmentRpc)),
		CanaryRpc: appointment.NewAppointment(zrpc.MustNewClient(c.AppointmentCanaryRpc)),
	}
}

func (s *ServiceContext) GetAppointmentRpcClient(ctx context.Context) appointment.Appointment {
	if isCanary, _ := ctx.Value("x-canary-route").(bool); isCanary {
		return s.CanaryRpc
	}
	return s.StableRpc
}
