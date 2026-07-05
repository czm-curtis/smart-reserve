// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/config"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/appointment"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config         config.Config
	AppointmentRpc appointment.Appointment
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		AppointmentRpc: appointment.NewAppointment(zrpc.MustNewClient(c.AppointmentRpc)),
	}
}
