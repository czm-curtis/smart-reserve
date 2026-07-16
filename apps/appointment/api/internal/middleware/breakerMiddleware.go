package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// responseWriter 包装 http.ResponseWriter 以捕获 HTTP 状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// BreakerState 熔断器状态
type BreakerState int32

const (
	BreakerClosed   BreakerState = iota // 闭合：正常通行
	BreakerOpen                         // 断开：快速失败
	BreakerHalfOpen                     // 半开：试探恢复
)

// BreakerMiddleware 网关层熔断中间件
// 监控下游 RPC 服务的错误率，超过阈值时自动熔断，保护系统不雪崩
type BreakerMiddleware struct {
	state        atomic.Int32
	failures     atomic.Int64
	successes    atomic.Int64
	lastFailTime atomic.Int64
	mu           sync.Mutex

	threshold int64         // 连续失败阈值
	cooldown  time.Duration // 熔断冷却时间
}

// NewBreakerMiddleware 创建熔断中间件
func NewBreakerMiddleware(threshold int64, cooldown time.Duration) *BreakerMiddleware {
	b := &BreakerMiddleware{
		threshold: threshold,
		cooldown:  cooldown,
	}
	b.state.Store(int32(BreakerClosed))
	return b
}

// Handle 熔断中间件处理逻辑
func (m *BreakerMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 跳过管理端点的熔断检查
		if isAdminPath(r.URL.Path) {
			next(w, r)
			return
		}

		// 检查熔断器状态
		if !m.allow() {
			logx.Errorf("🔴 [网关熔断] 熔断器已打开，拒绝请求: %s", r.URL.Path)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    503,
				"msg":     "服务暂时不可用，系统已启动熔断保护，请稍后再试",
				"orderNo": "",
			})
			return
		}

		// 包装 ResponseWriter 以捕获状态码
		crw := newResponseWriter(w)
		next(crw, r)

		// 根据响应状态码上报熔断器
		if crw.statusCode >= 500 {
			m.failure()
		} else {
			m.success()
		}
	}
}

func (m *BreakerMiddleware) allow() bool {
	state := BreakerState(m.state.Load())

	switch state {
	case BreakerClosed:
		return true

	case BreakerOpen:
		lastFail := time.Unix(0, m.lastFailTime.Load())
		if time.Since(lastFail) > m.cooldown {
			if m.state.CompareAndSwap(int32(BreakerOpen), int32(BreakerHalfOpen)) {
				logx.Infof("🔶 [网关熔断] OPEN → HALF_OPEN，试探恢复中...")
				return true
			}
		}
		return false

	case BreakerHalfOpen:
		if m.mu.TryLock() {
			return true
		}
		return false
	}

	return false
}

func (m *BreakerMiddleware) success() {
	m.failures.Store(0)
	state := BreakerState(m.state.Load())

	if state == BreakerHalfOpen {
		m.mu.Unlock()
		m.state.Store(int32(BreakerClosed))
		m.successes.Add(1)
		logx.Infof("🟢 [网关熔断] HALF_OPEN → CLOSED，网关已恢复！")
		return
	}

	m.successes.Add(1)
}

func (m *BreakerMiddleware) failure() {
	m.lastFailTime.Store(time.Now().UnixNano())
	state := BreakerState(m.state.Load())

	if state == BreakerHalfOpen {
		m.mu.Unlock()
		m.state.Store(int32(BreakerOpen))
		logx.Errorf("🔴 [网关熔断] HALF_OPEN → OPEN，试探失败，重新熔断！")
		return
	}

	if state == BreakerClosed {
		f := m.failures.Add(1)
		if f >= m.threshold {
			m.state.Store(int32(BreakerOpen))
			logx.Errorf("🔴 [网关熔断] CLOSED → OPEN！连续失败 %d 次，触发网关熔断！", f)
		}
	}
}

// State 获取当前熔断器状态（供管理端点查询）
func (m *BreakerMiddleware) State() BreakerState {
	return BreakerState(m.state.Load())
}

// Stats 获取统计信息
func (m *BreakerMiddleware) Stats() map[string]interface{} {
	stateStr := "CLOSED"
	switch m.State() {
	case BreakerOpen:
		stateStr = "OPEN"
	case BreakerHalfOpen:
		stateStr = "HALF_OPEN"
	}
	return map[string]interface{}{
		"state":     stateStr,
		"failures":  m.failures.Load(),
		"successes": m.successes.Load(),
		"threshold": m.threshold,
		"cooldown":  m.cooldown.String(),
	}
}

// Reset 重置熔断器
func (m *BreakerMiddleware) Reset() {
	m.failures.Store(0)
	m.successes.Store(0)
	m.state.Store(int32(BreakerClosed))
	logx.Infof("🔄 [网关熔断] 已被手动重置为 CLOSED")
}

func isAdminPath(path string) bool {
	return len(path) >= 5 && path[:5] == "/v1/a"
}
