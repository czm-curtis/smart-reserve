// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	AppointmentRpc zrpc.RpcClientConf // 告诉 Go 怎么解析 yaml 里的 RPC 地址
}
