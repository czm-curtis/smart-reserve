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

type PreloadScheduleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreloadScheduleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreloadScheduleLogic {
	return &PreloadScheduleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreloadScheduleLogic) PreloadSchedule(req *types.PreloadReq) (resp *types.PreloadResp, err error) {
	rpcResp, err := l.svcCtx.AppointmentRpc.PreloadSchedule(l.ctx, &pb.PreloadScheduleReq{
		ScheduleId: req.ScheduleId,
	})
	if err != nil {
		return nil, err
	}

	return &types.PreloadResp{
		Code: rpcResp.Code,
		Msg:  rpcResp.Msg,
	}, nil
}
