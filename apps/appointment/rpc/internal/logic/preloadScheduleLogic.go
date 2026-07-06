package logic

import (
	"context"
	"fmt"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/svc"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreloadScheduleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreloadScheduleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreloadScheduleLogic {
	return &PreloadScheduleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PreloadScheduleLogic) PreloadSchedule(in *pb.PreloadScheduleReq) (*pb.PreloadScheduleResp, error) {
	l.Infof("【缓存预热触发】正在预热场次 ID: %d", in.ScheduleId)
	schedule, err := l.svcCtx.ScheduleModel.FindOne(l.ctx, in.ScheduleId)
	if err == sqlx.ErrNotFound {
		l.Errorf("预热失败：MySQL 中不存在该场次 ID: %d，触发防穿透保护", in.ScheduleId)
		return &pb.PreloadScheduleResp{
			Code: 4004,
			Msg:  "预热场次不存在",
		}, nil
	} else if err != nil {
		l.Errorf("MySQL 查询异常: %v", err)
		return nil, err
	}
	redisKey := fmt.Sprintf("reserve:slots:%d", schedule.Id)
	err = l.svcCtx.RedisClient.Set(redisKey, fmt.Sprintf("%d", schedule.TotalSlots))
	if err != nil {
		l.Errorf("Redis 写入失败: %v", err)
		return nil, err
	}

	l.Infof("【缓存预热成功】🌟 场次《%s》已成功推入 Redis 阵地！库存: %d", schedule.Title, schedule.TotalSlots)

	return &pb.PreloadScheduleResp{
		Code: 0,
		Msg:  "缓存预热成功",
	}, nil
}
