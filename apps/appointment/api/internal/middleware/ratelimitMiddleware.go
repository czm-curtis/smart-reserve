package middleware

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RateLimitMiddleware 自适应双层限流中间件
// 第一层：IP 级别限流 — 防止单一 IP 高频刷接口（反黄牛）
// 第二层：用户级别限流 — 防止单用户高频刷接口（反脚本）
type RateLimitMiddleware struct {
	ipLimiter   *limit.PeriodLimit // IP 级别限流器
	userLimiter *limit.PeriodLimit // 用户级别限流器
}

// NewRateLimitMiddleware 创建限流中间件
// period: 限流周期（秒）
// ipQuota: IP 级别每周期最大请求数
// userQuota: 用户级别每周期最大请求数（建议设为 IP 级别的 2 倍）
func NewRateLimitMiddleware(redisConf redis.RedisConf, period, ipQuota, userQuota int) *RateLimitMiddleware {
	store := redis.MustNewRedis(redisConf)

	return &RateLimitMiddleware{
		ipLimiter:   limit.NewPeriodLimit(period, ipQuota, store, "ratelimit:ip"),
		userLimiter: limit.NewPeriodLimit(period, userQuota, store, "ratelimit:user"),
	}
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 获取客户端 IP
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		// 2. IP 级别限流判定
		code, err := m.ipLimiter.Take(ip)
		if err != nil {
			writeRateLimitError(w, "系统繁忙，请稍后再试")
			return
		}
		if code == limit.OverQuota {
			writeRateLimitError(w, "抢购太火爆了！您的IP请求过于频繁，请稍后再试")
			return
		}

		// 3. 用户级别限流判定
		userId := r.Header.Get("X-User-Id")
		if userId != "" {
			code, err := m.userLimiter.Take(userId)
			if err != nil {
				writeRateLimitError(w, "系统繁忙，请稍后再试")
				return
			}
			if code == limit.OverQuota {
				writeRateLimitError(w, "操作太快了！请放慢速度，稍后再试")
				return
			}
		}

		// 4. 通过限流，继续处理
		next.ServeHTTP(w, r)
	}
}

func writeRateLimitError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 429,
		"msg":  msg,
	})
}
