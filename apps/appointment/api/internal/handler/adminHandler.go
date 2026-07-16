package handler

import (
	"encoding/json"
	"net/http"

	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/middleware"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/svc"
)

const (
	// Redis 中模拟 Kafka 故障的标记 Key
	simulateKey = "simulate:kafka:failure"
)

// AdminHandler 管理端点处理器
// 提供治理状态查看、故障模拟等功能，用于演示微服务治理能力
type AdminHandler struct {
	svcCtx    *svc.ServiceContext
	breakerMw *middleware.BreakerMiddleware
}

func NewAdminHandler(svcCtx *svc.ServiceContext, breakerMw *middleware.BreakerMiddleware) *AdminHandler {
	return &AdminHandler{svcCtx: svcCtx, breakerMw: breakerMw}
}

// RegisterAdminRoutes 注册管理端点路由
func RegisterAdminRoutes(mux *http.ServeMux, svcCtx *svc.ServiceContext, breakerMw *middleware.BreakerMiddleware) {
	h := NewAdminHandler(svcCtx, breakerMw)

	// GET /v1/admin/status — 查看系统治理状态
	mux.HandleFunc("/v1/admin/status", h.Status)

	// POST /v1/admin/simulate/failure — 模拟 Kafka 故障（触发熔断降级演示）
	mux.HandleFunc("/v1/admin/simulate/failure", h.SimulateFailure)

	// POST /v1/admin/simulate/recovery — 从故障中恢复
	mux.HandleFunc("/v1/admin/simulate/recovery", h.SimulateRecovery)

	// POST /v1/admin/breaker/reset — 重置网关熔断器
	mux.HandleFunc("/v1/admin/breaker/reset", h.ResetBreaker)
}

// Status 返回系统治理状态
func (h *AdminHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 检查模拟开关状态
	isSimulating := h.isSimulating()

	// 查询降级队列长度
	queueLen, _ := h.svcCtx.RedisClient.LlenCtx(r.Context(), "degradation:orders")

	resp := map[string]interface{}{
		"gateway": map[string]interface{}{
			"breaker": h.breakerMw.Stats(),
		},
		"kafka": map[string]interface{}{
			"simulatingFailure": isSimulating,
		},
		"degradation": map[string]interface{}{
			"queueKey": "degradation:orders",
			"queueLen": queueLen,
		},
		"tip": "POST /v1/admin/simulate/failure 模拟 Kafka 故障 → 观察熔断降级",
	}

	json.NewEncoder(w).Encode(resp)
}

// SimulateFailure 模拟 Kafka 写入故障
func (h *AdminHandler) SimulateFailure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 通过 Redis 写入模拟标记，所有 RPC 实例都能感知
	h.svcCtx.RedisClient.SetCtx(r.Context(), simulateKey, "1")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":   0,
		"msg":    "故障模拟已开启 — Kafka 写入将返回失败，连续 3 次失败后触发熔断",
		"action": "现在发送抢票请求观察降级效果（返回'降级模式:订单排队处理中'）",
		"recovery": "POST /v1/admin/simulate/recovery 恢复 Kafka",
	})
}

// SimulateRecovery 停止故障模拟，恢复 Kafka
func (h *AdminHandler) SimulateRecovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 删除模拟标记
	h.svcCtx.RedisClient.DelCtx(r.Context(), simulateKey)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  "故障模拟已关闭 — Kafka 恢复正常，等待熔断器冷却(30s)后自动补偿降级订单",
		"hint": "降级补偿 Worker 会在熔断器恢复(CLOSED)后自动将 Redis 队列中的订单重新投递 Kafka",
	})
}

// ResetBreaker 重置网关熔断器
func (h *AdminHandler) ResetBreaker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	h.breakerMw.Reset()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  "网关熔断器已手动重置为 CLOSED",
	})
}

func (h *AdminHandler) isSimulating() bool {
	val, err := h.svcCtx.RedisClient.Get(simulateKey)
	if err != nil {
		return false
	}
	return val == "1"
}

