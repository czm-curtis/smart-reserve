package breaker

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	// DegradationListKey Redis 降级队列 Key
	DegradationListKey = "degradation:orders"
)

// DegradationQueue Redis 降级队列
// 当 Kafka 熔断时，预约订单降级写入此队列，等待异步补偿
type DegradationQueue struct {
	redis *redis.Redis
}

// NewDegradationQueue 创建降级队列
func NewDegradationQueue(rc *redis.Redis) *DegradationQueue {
	return &DegradationQueue{redis: rc}
}

// Enqueue 将消息写入降级队列（左进）
func (d *DegradationQueue) Enqueue(ctx context.Context, msg string) error {
	_, err := d.redis.LpushCtx(ctx, DegradationListKey, msg)
	if err != nil {
		logx.Errorf("❌ [降级队列] 入队失败: %v, msg: %s", err, msg)
		return fmt.Errorf("degradation enqueue failed: %w", err)
	}
	logx.Infof("📦 [降级队列] 订单已降级存储: %s", msg)
	return nil
}

// Dequeue 从降级队列取出消息（右出，FIFO）
func (d *DegradationQueue) Dequeue(ctx context.Context) (string, error) {
	msg, err := d.redis.RpopCtx(ctx, DegradationListKey)
	if err != nil {
		return "", err
	}
	return msg, nil
}

// Len 获取降级队列长度
func (d *DegradationQueue) Len(ctx context.Context) (int64, error) {
	result, err := d.redis.LlenCtx(ctx, DegradationListKey)
	if err != nil {
		return 0, err
	}
	return int64(result), nil
}

// Drain 补偿投递：遍历降级队列，将消息重新投递到 Kafka
// 返回成功补偿的数量
func (d *DegradationQueue) Drain(ctx context.Context, pusher func(context.Context, string) error) int {
	compensated := 0
	for {
		msg, err := d.Dequeue(ctx)
		if err != nil || msg == "" {
			break
		}

		if pushErr := pusher(ctx, msg); pushErr != nil {
			// 投递失败，放回队列头部
			_ = d.Enqueue(ctx, msg)
			logx.Errorf("⚠️ [降级补偿] 投递失败，放回队列: %v", pushErr)
			break
		}

		compensated++
		logx.Infof("✅ [降级补偿] 成功补偿投递: %s", msg)
	}
	return compensated
}
