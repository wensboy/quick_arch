package model

import (
	"net/http"

	errs "github.com/wensboy/quick_arch/internal/error"
)

// StatusOf 返回错误对应的 HTTP 状态码: 业务错误按码映射, 内部错误一律 500.
func StatusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}

	e, ok := errs.FromError(err)
	if !ok || e.Category() != errs.CategoryBusiness {
		return http.StatusInternalServerError
	}
	if status, ok := businessStatus[e.Code()]; ok {
		return status
	}
	return http.StatusBadRequest
}

// businessStatus 是业务错误码到 HTTP 状态码的映射, 未登记的业务错误按 400 处理.
var businessStatus = map[errs.Code]int{
	errs.ErrInvalidParam.Code: http.StatusBadRequest,
	errs.ErrPermission.Code:   http.StatusForbidden,
	errs.ErrNotFound.Code:     http.StatusNotFound,
	errs.ErrConflict.Code:     http.StatusConflict,
}
