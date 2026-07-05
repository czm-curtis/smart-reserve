package svc

import (
	"context"
	"time"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config      config.Config
	RedisClient *redis.Redis
	OrderChan   chan *pb.CreateAppointmentReq
}

func NewServiceContext(c config.Config) *ServiceContext {
	rc := redis.MustNewRedis(c.BizRedis)
	// =================【架构师级：依赖预热逻辑】=================
	// 初始化带有 10000 缓冲区的通道，防止高并发时瞬间挤爆内存
	orderChan := make(chan *pb.CreateAppointmentReq, 1000)

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
		go startOrderConsumer(i, orderChan)
	}
	return &ServiceContext{
		Config:      c,
		RedisClient: rc,
		OrderChan:   orderChan,
	}
}

func startOrderConsumer(workerId int, ch chan *pb.CreateAppointmentReq) {
	// 使用 range 持续阻塞监听通道，直到通道被关闭
	for req := range ch {
		// 模拟真实数据库（MySQL）写入的缓慢开销
		// 磁盘 I/O 极其昂贵，我们人为让它卡顿 100 毫秒，用来模拟真实高并发下的落库瓶颈
		time.Sleep(1000 * time.Millisecond)
		// 打印落库成功日志
		logx.Infof("【工人 %d 号】成功将预约数据持久化到 MySQL！用户ID: %d, 场次ID: %d",
			workerId, req.UserId, req.ScheduleId)
	}
}
