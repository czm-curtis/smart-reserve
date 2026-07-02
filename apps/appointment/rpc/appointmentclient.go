package main

import (
	"flag"
	"fmt"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/appointmentclient"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/server"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/appointmentclient.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		appointmentclient.RegisterAppointmentServer(grpcServer, server.NewAppointmentServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
