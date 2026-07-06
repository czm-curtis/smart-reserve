package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/model"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config                config.Config
	RedisClient           *redis.Redis
	OrderChan             chan *pb.CreateAppointmentReq
	AppointmentOrderModel model.AppointmentOrderModel
	ScheduleModel         model.ScheduleModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	rc := redis.MustNewRedis(c.BizRedis)
	sqlConn := sqlx.NewMysql(c.DataSource)
	orderModel := model.NewAppointmentOrderModel(sqlConn, cache.CacheConf{
		{RedisConf: c.BizRedis, Weight: 100},
	})
	scheduleModel := model.NewScheduleModel(sqlConn, cache.CacheConf{
		{RedisConf: c.BizRedis, Weight: 100},
	})
	// 初始化带有 10000 缓冲区的通道，防止高并发时瞬间挤爆内存
	orderChan := make(chan *pb.CreateAppointmentReq, 1000)

	// =================【架构师级：依赖预热逻辑】=================
	// 在服务初始化时，主动拉起底层连接，防止第一波高并发请求遭遇 TCP 握手长尾
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// 并发发出 Ping，迫使连接池内部建立并维持活跃的 TCP 长连接
		for i := 0; i < 10; i++ {
			go rc.PingCtx(ctx)
		}
	}()

	// 2. 【新增】：拉起后台异步落库消费者（启动 5 个并发工人）
	for i := 0; i < 5; i++ {
		go startOrderConsumer(i, orderChan, orderModel)
	}
	return &ServiceContext{
		Config:                c,
		RedisClient:           rc,
		OrderChan:             orderChan,
		AppointmentOrderModel: orderModel,
		ScheduleModel:         scheduleModel,
	}
}

func startOrderConsumer(workerId int, ch chan *pb.CreateAppointmentReq, m model.AppointmentOrderModel) {
	// 使用 range 持续阻塞监听通道，直到通道被关闭
	for req := range ch {
		orderNo := fmt.Sprintf("RES_%s_%d", time.Now().Format("20060102"), req.UserId)
		_, err := m.Insert(context.Background(), &model.AppointmentOrder{
			UserId:     uint64(req.UserId),
			ScheduleId: uint64(req.ScheduleId),
			OrderNo:    orderNo,
			Status:     1, // 成功
		})
		// 打印落库成功日志
		if err != nil {
			logx.Errorf("【工人 %d 号】MySQL 写入失败！！错误: %v, 用户ID: %d", workerId, err, req.UserId)
			continue
		}

		logx.Infof("【工人 %d 号】🌟 真实 MySQL 写入成功！用户ID: %d, 单号: %s", workerId, req.UserId, orderNo)
	}
}
