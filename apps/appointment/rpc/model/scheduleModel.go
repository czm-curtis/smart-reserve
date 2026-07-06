package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ScheduleModel = (*customScheduleModel)(nil)

type (
	// ScheduleModel is an interface to be customized, add more methods here,
	// and implement the added methods in customScheduleModel.
	ScheduleModel interface {
		scheduleModel
	}

	customScheduleModel struct {
		*defaultScheduleModel
	}
)

// NewScheduleModel returns a model for the database table.
func NewScheduleModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ScheduleModel {
	return &customScheduleModel{
		defaultScheduleModel: newScheduleModel(conn, c, opts...),
	}
}
