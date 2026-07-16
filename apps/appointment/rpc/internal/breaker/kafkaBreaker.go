package breaker

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// State 熔断器状态
type State int32

const (
	StateClosed   State = iota // 闭合：正常通行
	StateOpen                  // 断开：快速失败
	StateHalfOpen              // 半开：试探恢复
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// KafkaBreaker Kafka 写入熔断器
// 核心逻辑：
//   - Closed: 正常放行，连续失败 >= threshold 时 → Open
//   - Open: 直接拒绝，等待 cooldown 后 → HalfOpen
//   - HalfOpen: 允许少数试探请求，成功则 → Closed，失败则 → Open
type KafkaBreaker struct {
	state        atomic.Int32
	failures     atomic.Int64
	successes    atomic.Int64
	lastFailTime atomic.Int64 // unix nano
	threshold    int64
	cooldown     time.Duration
	mu           sync.Mutex
}

// NewKafkaBreaker 创建熔断器
// threshold: 连续失败多少次后熔断
// cooldown: 熔断后等待多久进入半开状态
func NewKafkaBreaker(threshold int64, cooldown time.Duration) *KafkaBreaker {
	b := &KafkaBreaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
	b.state.Store(int32(StateClosed))
	return b
}

// Allow 判断是否允许通过
func (b *KafkaBreaker) Allow() bool {
	state := State(b.state.Load())

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		lastFail := time.Unix(0, b.lastFailTime.Load())
		if time.Since(lastFail) > b.cooldown {
			// 冷却期已过，尝试进入半开状态
			if b.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
				logx.Infof("🔶 [熔断器] OPEN → HALF_OPEN，开始试探恢复...")
				return true
			}
		}
		return false

	case StateHalfOpen:
		// 半开状态只放行一个试探请求
		if b.mu.TryLock() {
			return true
		}
		return false
	}

	return false
}

// Success 上报成功，关闭熔断器
func (b *KafkaBreaker) Success() {
	b.failures.Store(0)
	state := State(b.state.Load())

	if state == StateHalfOpen {
		b.mu.Unlock()
		b.state.Store(int32(StateClosed))
		b.successes.Add(1)
		logx.Infof("🟢 [熔断器] HALF_OPEN → CLOSED，服务已恢复正常！")
		return
	}

	b.successes.Add(1)
}

// Failure 上报失败
func (b *KafkaBreaker) Failure() {
	b.lastFailTime.Store(time.Now().UnixNano())
	state := State(b.state.Load())

	if state == StateHalfOpen {
		// 半开试探失败，重新断开
		b.mu.Unlock()
		b.state.Store(int32(StateOpen))
		logx.Errorf("🔴 [熔断器] HALF_OPEN → OPEN，试探请求失败，重新熔断！")
		return
	}

	if state == StateClosed {
		f := b.failures.Add(1)
		if f >= b.threshold {
			b.state.Store(int32(StateOpen))
			logx.Errorf("🔴 [熔断器] CLOSED → OPEN！连续失败 %d 次，触发熔断保护！", f)
		}
	}
}

// GetState 获取当前状态
func (b *KafkaBreaker) GetState() State {
	return State(b.state.Load())
}

// Stats 获取统计信息
func (b *KafkaBreaker) Stats() map[string]interface{} {
	return map[string]interface{}{
		"state":     b.GetState().String(),
		"failures":  b.failures.Load(),
		"successes": b.successes.Load(),
		"threshold": b.threshold,
		"cooldown":  b.cooldown.String(),
	}
}

// Reset 重置熔断器（用于模拟恢复）
func (b *KafkaBreaker) Reset() {
	b.failures.Store(0)
	b.successes.Store(0)
	b.state.Store(int32(StateClosed))
	logx.Infof("🔄 [熔断器] 已被手动重置为 CLOSED")
}
