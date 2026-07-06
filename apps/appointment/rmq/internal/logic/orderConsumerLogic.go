package logic

import (
	"context"
	"encoding/json"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rmq/internal/svc"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/model"
	"github.com/zeromicro/go-zero/core/logx"
)

type OrderConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOrderConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *OrderConsumer {
	return &OrderConsumer{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Kafka 消息的 JSON 解析 DTO
type KafkaOrderMsg struct {
	UserId     uint64 `json:"userId"`
	ScheduleId uint64 `json:"scheduleId"`
	OrderNo    string `json:"orderNo"`
}

// Consume 【核心方法】：监听 Kafka 队列的真工人
func (c *OrderConsumer) Consume(ctx context.Context, key, val string) error {
	c.Infof("【Kafka 管道接收消息】key: %s, payload: %s", key, val)

	// 1. 反序列化消息
	var msg KafkaOrderMsg
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		c.Errorf("消息格式极其差劲，反序列化失败: %v。予以抛弃保护队列！", err)
		return nil // 遇到脏消息返回 nil 提交 ACK，防止坏消息卡死队列
	}

	// 2. 真正安全、持久化地写入 MySQL
	_, err := c.svcCtx.AppointmentOrderModel.Insert(ctx, &model.AppointmentOrder{
		UserId:     msg.UserId,
		ScheduleId: msg.ScheduleId,
		OrderNo:    msg.OrderNo,
		Status:     1, // 1: 预约成功
	})

	if err != nil {
		c.Errorf("【严重报警】MySQL 持久化落库失败！单号: %s, 错误: %v", msg.OrderNo, err)
		return err // 【核心】：返回错误，不提交 ACK，Kafka 随后将触发重试流，确保资产不丢失！
	}

	c.Infof("【Kafka 消费成功】🌟 真实订单已安全、永久落地 MySQL 磁盘！流水单号: %s", msg.OrderNo)
	return nil
}
