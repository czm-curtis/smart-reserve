package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/internal/svc"
	"github.com/czm-curtis/smart-reserve/apps/appointment/rpc/pb"
	"google.golang.org/grpc/metadata"

	"github.com/zeromicro/go-zero/core/logx"
)

// 定义 Redis Lua 脚本：将防重和扣减名额合二为一
// KEYS[1]: 名额库存 Key (e.g., reserve:slots:scheduleId)
// KEYS[2]: 用户防重 Key (e.g., reserve:user:userId:scheduleId)
const luaReserveScript = `
    -- 1. 校验是否重复预约
    local isBooked = redis.call('GET', KEYS[2])
    if isBooked then
        return -1 -- 代表已存在预约记录（重复预约）
    end

    -- 2. 校验名额库存是否足够
    local count = tonumber(redis.call('GET', KEYS[1]))
    if not count or count <= 0 then
        return -2 -- 代表名额不足（爆仓/超卖）
    end

    -- 3. 执行原子扣减名额
    redis.call('DECR', KEYS[1])
    
    -- 4. 记录用户防重标记，设置过期时间（比如 24 小时 = 86400 秒），防止占坑不拉
    redis.call('SET', KEYS[2], '1', 'EX', 86400)

    return 1 -- 代表全部成功
`

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

func (l *CreateAppointmentLogic) CreateAppointment(in *pb.CreateAppointmentReq) (*pb.CreateAppointmentResp, error) {
	slotsKey := fmt.Sprintf("reserve:slots:%d", in.ScheduleId)
	userKey := fmt.Sprintf("reserve:user:%d:%d", in.UserId, in.ScheduleId)
	md, ok := metadata.FromIncomingContext(l.ctx)
	isCanary := false
	if ok {
		if values := md.Get("x-canary-stain"); len(values) > 0 && values[0] == "true" {
			isCanary = true
		}
	}
	// 2. 打印具有明显特征的日志，方便我们肉眼观察分流
	if isCanary {
		l.Logger.Infof("🔥 [Canary Cluster 9091] 命中灰度！正在处理用户 %d 的预约...", in.UserId)

		// 你甚至可以在灰度环境返回稍微不一样的 Msg，方便 K6 验证
		return &pb.CreateAppointmentResp{
			Code:    200,
			Msg:     "[Canary] 预约成功",
			OrderNo: "CANARY_ORDER_xxx",
		}, nil
	}
	// 3. 正常 Stable 流程
	l.Logger.Infof("🟢 [Stable Cluster 9090] 正常流量。正在处理用户 %d 的预约...", in.UserId)
	res, err := l.svcCtx.RedisClient.EvalCtx(l.ctx, luaReserveScript, []string{slotsKey, userKey})
	if err != nil {
		l.Error("Redis Lua 脚本执行失败: %v", err)
		return &pb.CreateAppointmentResp{Code: 500, Msg: "系统繁忙，请稍后再试"}, nil
	}
	// 解析 Lua 脚本返回的结果值
	code := res.(int64)
	switch code {
	case -1:
		l.Errorf("用户 %d 重复预约场次 %d", in.UserId, in.ScheduleId)
		return &pb.CreateAppointmentResp{
			Code: 4001,
			Msg:  "您已预约过该场次，请勿重复预约",
		}, nil
	case -2:
		l.Errorf("场次 %d 名额已满", in.ScheduleId)
		return &pb.CreateAppointmentResp{
			Code: 4002,
			Msg:  "很抱歉，该时段预约名额已满",
		}, nil
	case 1:
		// 1. 扣减成功，生成唯一的预约流水号
		orderNo := fmt.Sprintf("RES_%s_%d", time.Now().Format("20060102"), in.UserId)
		l.Infof("【预约成功】用户: %d, 场次: %d, 单号: %s", in.UserId, in.ScheduleId, orderNo)

		// 2. 【核心改动】：构造 JSON 消息体，解耦下游落库
		// 生产场景下，可以定义专用的消息 DTO，这里我们直接将核心要素序列化
		msgJson := fmt.Sprintf(`{"userId":%d,"scheduleId":%d,"orderNo":"%s"}`, in.UserId, in.ScheduleId, orderNo)
		// 3. 推送至 Kafka 管道（KqPusher 内部自带高并发缓冲与自动重试机制）
		err = l.svcCtx.KqPusherClient.Push(l.ctx, msgJson)
		if err != nil {
			// 如果 Kafka 挂了，这里记录严重警报，但这属于下游落库风险，名额已经由 Redis 锁死，不影响前端给用户返回成功
			l.Errorf("Kafka 消息投递严重失败！单号: %s, 错误: %v", orderNo, err)
		}

		return &pb.CreateAppointmentResp{
			Code:    0,
			Msg:     "预约成功",
			OrderNo: orderNo,
		}, nil
	}

	return &pb.CreateAppointmentResp{Code: 500, Msg: "未知系统错误"}, nil
}
