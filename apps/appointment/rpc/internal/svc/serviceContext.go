package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/breaker"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/model"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"github.com/zeromicro/go-queue/kq"
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
	KqPusherClient        *kq.Pusher

	// ==== 微服务治理组件 ====
	KafkaBreaker     *breaker.KafkaBreaker     // Kafka 写入熔断器
	DegradationQueue *breaker.DegradationQueue // Redis 降级队列
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
	pusherClient := kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic)
	orderChan := make(chan *pb.CreateAppointmentReq, 1000)

	// ================= 微服务治理：熔断与降级 =================
	kafkaBreaker := breaker.NewKafkaBreaker(
		c.KafkaBreaker.Threshold,
		c.KafkaBreaker.Cooldown,
	)
	degradationQueue := breaker.NewDegradationQueue(rc)

	// ================= 依赖预热逻辑 =================
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		for i := 0; i < 10; i++ {
			go rc.PingCtx(ctx)
		}
	}()

	// 后台异步落库消费者（5 个并发工人）
	for i := 0; i < 5; i++ {
		go startOrderConsumer(i, orderChan, orderModel)
	}

	// ================= 后台降级补偿 Worker =================
	// 定期检查熔断器状态，当恢复后自动补偿降级队列中的订单
	go startDegradationWorker(kafkaBreaker, degradationQueue, pusherClient)

	svcCtx := &ServiceContext{
		Config:                c,
		RedisClient:           rc,
		OrderChan:             orderChan,
		AppointmentOrderModel: orderModel,
		ScheduleModel:         scheduleModel,
		KqPusherClient:        pusherClient,
		KafkaBreaker:          kafkaBreaker,
		DegradationQueue:      degradationQueue,
	}

	logx.Infof("🛡️ [治理组件] Kafka 熔断器已就绪 (threshold=%d, cooldown=%s)", c.KafkaBreaker.Threshold, c.KafkaBreaker.Cooldown)
	logx.Infof("📦 [治理组件] Redis 降级队列已就绪 (key=%s)", breaker.DegradationListKey)

	return svcCtx
}

func startOrderConsumer(workerId int, ch chan *pb.CreateAppointmentReq, m model.AppointmentOrderModel) {
	for req := range ch {
		orderNo := fmt.Sprintf("RES_%s_%d", time.Now().Format("20060102"), req.UserId)
		_, err := m.Insert(context.Background(), &model.AppointmentOrder{
			UserId:     uint64(req.UserId),
			ScheduleId: uint64(req.ScheduleId),
			OrderNo:    orderNo,
			Status:     1,
		})
		if err != nil {
			logx.Errorf("【工人 %d 号】MySQL 写入失败！！错误: %v, 用户ID: %d", workerId, err, req.UserId)
			continue
		}
		logx.Infof("【工人 %d 号】🌟 真实 MySQL 写入成功！用户ID: %d, 单号: %s", workerId, req.UserId, orderNo)
	}
}

// startDegradationWorker 后台降级补偿 Worker
// 每 5 秒检查一次：如果熔断器已恢复（Closed），则尝试将降级队列中的订单重新投递到 Kafka
func startDegradationWorker(kb *breaker.KafkaBreaker, dq *breaker.DegradationQueue, pusher *kq.Pusher) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if kb.GetState() != breaker.StateClosed {
			continue // 熔断器未恢复，暂不补偿
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		compensated := dq.Drain(ctx, func(ctx context.Context, msg string) error {
			return pusher.Push(ctx, msg)
		})
		cancel()

		if compensated > 0 {
			logx.Infof("🔄 [降级补偿] 本轮成功补偿 %d 条订单", compensated)
		}
	}
}
