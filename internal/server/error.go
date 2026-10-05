package server

import (
	"net/http"

	"github.com/labstack/echo/v5"

	errs "github.com/wensboy/quick_arch/internal/error"
	"github.com/wensboy/quick_arch/model"
)

// handleHTTPError 是全局错误出口: 所有未处理错误统一输出 model.Response.
func handleHTTPError(c *echo.Context, err error) {
	if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
		return
	}

	status, body := failureOf(err)
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(status)
		return
	}
	_ = c.JSON(status, body)
}

// failureOf 把错误映射为状态码与响应体; 非标准化错误 (echo 路由/中间件) 沿用其状态码.
func failureOf(err error) (int, model.Response) {
	if _, ok := errs.FromError(err); ok {
		return model.StatusOf(err), model.Failure(err)
	}

	status := echo.StatusCode(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	def, ok := definitionOf(status)
	if !ok {
		def = errs.ErrInternal
	}
	return status, model.Failure(def)
}

// definitionOf 把 HTTP 状态码映射回标准错误定义.
func definitionOf(status int) (errs.Definition, bool) {
	switch status {
	case http.StatusBadRequest:
		return errs.ErrInvalidParam, true
	case http.StatusForbidden:
		return errs.ErrPermission, true
	case http.StatusNotFound:
		return errs.ErrNotFound, true
	case http.StatusConflict:
		return errs.ErrConflict, true
	case http.StatusInternalServerError:
		return errs.ErrInternal, true
	default:
		return errs.Definition{}, false
	}
}
