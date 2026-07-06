package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	KqConsumerConf kq.KqConf
	DataSource     string
	BizRedis       redis.RedisConf
}
