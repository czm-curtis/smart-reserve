package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rmq/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rmq/internal/logic"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rmq/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/rmq.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	orderConsumer := logic.NewOrderConsumer(ctx, svcCtx)

	// 创建真正具备高并发负载均衡能力的 Kafka 消费者队列
	q := kq.MustNewQueue(c.KqConsumerConf, kq.WithHandle(orderConsumer.Consume))
	defer q.Stop()

	fmt.Printf("Starting independent Kafka consumer server at topic: %s...\n", c.KqConsumerConf.Topic)

	// 使用 Go-zero 服务群组将其拉起
	serviceGroup := service.NewServiceGroup()
	serviceGroup.Add(q)
	serviceGroup.Start()
}
