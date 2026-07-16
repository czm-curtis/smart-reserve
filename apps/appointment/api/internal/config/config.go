// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// RateLimitConf 限流配置
type RateLimitConf struct {
	Period    int `json:"period" default:"1"`
	IpQuota   int `json:"ipQuota" default:"5"`
	UserQuota int `json:"userQuota" default:"10"`
}

// BreakerConf 熔断配置
type BreakerConf struct {
	Threshold int64         `json:"threshold" default:"5"`
	Cooldown  time.Duration `json:"cooldown" default:"30s"`
}

type Config struct {
	rest.RestConf
	BizRedis             redis.RedisConf    // 用于限流和通用缓存
	AppointmentRpc       zrpc.RpcClientConf // 稳定版 RPC
	AppointmentCanaryRpc zrpc.RpcClientConf // 灰度版 RPC
	RateLimit            RateLimitConf      `json:",optional"` // 限流配置
	GatewayBreaker       BreakerConf        `json:",optional"` // 网关熔断配置
}
