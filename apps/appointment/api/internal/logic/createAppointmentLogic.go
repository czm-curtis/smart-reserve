// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package logic

import (
	"context"

	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/svc"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/types"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAppointmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAppointmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAppointmentLogic {
	return &CreateAppointmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAppointmentLogic) CreateAppointment(req *types.AppointmentReq) (resp *types.AppointmentResp, err error) {
	rpcResp, err := l.svcCtx.AppointmentRpc.CreateAppointment(l.ctx, &pb.CreateAppointmentReq{
		UserId:     req.UserId,
		ScheduleId: req.ScheduleId,
	})
	if err != nil {
		return nil, err
	}
	return &types.AppointmentResp{
		Code:    int(rpcResp.Code),
		Msg:     rpcResp.Msg,
		OrderNo: rpcResp.OrderNo,
	}, nil
}
