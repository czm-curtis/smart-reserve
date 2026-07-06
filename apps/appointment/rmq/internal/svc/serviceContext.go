package svc

import (
	"github.com/czm-curtis/smart-reserve/apps/appointment/rmq/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/model" // 优雅地跨模块引入
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config                config.Config
	AppointmentOrderModel model.AppointmentOrderModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DataSource)

	return &ServiceContext{
		Config: c,
		// 初始化 Model 层，RMQ 服务正式具备写入 MySQL 的能力
		AppointmentOrderModel: model.NewAppointmentOrderModel(sqlConn, cache.CacheConf{
			{RedisConf: c.BizRedis, Weight: 100},
		}),
	}
}
