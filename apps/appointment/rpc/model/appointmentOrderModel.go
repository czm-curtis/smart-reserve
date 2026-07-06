package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AppointmentOrderModel = (*customAppointmentOrderModel)(nil)

type (
	// AppointmentOrderModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAppointmentOrderModel.
	AppointmentOrderModel interface {
		appointmentOrderModel
	}

	customAppointmentOrderModel struct {
		*defaultAppointmentOrderModel
	}
)

// NewAppointmentOrderModel returns a model for the database table.
func NewAppointmentOrderModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AppointmentOrderModel {
	return &customAppointmentOrderModel{
		defaultAppointmentOrderModel: newAppointmentOrderModel(conn, c, opts...),
	}
}
