package middleware

import (
	"context"
	"net/http"
	"strconv"

	"google.golang.org/grpc/metadata"
)

type CanaryMiddleware struct{}

func NewCanaryMiddleware() *CanaryMiddleware {
	return &CanaryMiddleware{}
}

func (m *CanaryMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 从 HTTP Header 中读取用户 ID (后续可以在 K6 脚本中传入)
		userIdStr := r.Header.Get("X-User-Id")
		userId, _ := strconv.Atoi(userIdStr)

		// 核心染色逻辑：UserId 末尾是 0 的用户（占 10% 概率）走灰度
		if userId > 0 && userId%10 == 0 {
			// 1. 本地进程内染色标签
			ctx = context.WithValue(ctx, "x-canary-route", true)

			// 2. 跨进程 gRPC Metadata 染色标签（向下游 RPC 广播）
			md := metadata.Pairs("x-canary-stain", "true")
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
