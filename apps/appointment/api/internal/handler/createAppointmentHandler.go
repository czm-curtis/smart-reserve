// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package handler

import (
	"net/http"

	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/logic"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/svc"
	"github.com/czm-curtis/smart-reserve/apps/appointment/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateAppointmentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppointmentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCreateAppointmentLogic(r.Context(), svcCtx)
		resp, err := l.CreateAppointment(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
