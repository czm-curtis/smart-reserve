package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

// BreakerConf 熔断器配置
type BreakerConf struct {
	Threshold int64         `json:"threshold" default:"3"`
	Cooldown  time.Duration `json:"cooldown" default:"30s"`
}

type Config struct {
	zrpc.RpcServerConf
	BizRedis     redis.RedisConf
	DataSource   string
	KqPusherConf struct {
		Brokers []string
		Topic   string
	}
	KafkaBreaker BreakerConf `json:",optional"` // Kafka 写入熔断器配置
}
