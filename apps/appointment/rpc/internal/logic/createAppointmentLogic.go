package logic

import (
	"context"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/appointmentclient"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAppointmentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAppointmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAppointmentLogic {
	return &CreateAppointmentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateAppointmentLogic) CreateAppointment(in *appointmentclient.CreateAppointmentReq) (*appointmentclient.CreateAppointmentResp, error) {
	// todo: add your logic here and delete this line

	return &appointmentclient.CreateAppointmentResp{}, nil
}
