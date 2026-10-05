package model

import (
	"encoding/json"
	"errors"

	errs "github.com/wensboy/quick_arch/internal/error"
)

// CodeSuccess 代表成功; 失败时业务错误取正码, 内部错误取负码.
const CodeSuccess int32 = 0

// Success 构造成功响应, message 固定为空.
func Success(data any) Response {
	return Response{Code: CodeSuccess, Data: data}
}

// Failure 依据错误构造失败响应: 业务错误保留具体消息, 其余分类只回默认文案.
// 既接受 *errs.Error, 也接受裸的 errs.Definition.
func Failure(err error) Response {
	if err == nil {
		return Success(nil)
	}

	if e, ok := errs.FromError(err); ok {
		if e.Category() == errs.CategoryBusiness {
			return newFailure(e.Definition(), e.Message())
		}
		return newFailure(e.Definition(), e.Definition().Message)
	}

	var def errs.Definition
	if errors.As(err, &def) {
		return newFailure(def, def.Message)
	}
	return newFailure(errs.ErrInternal, errs.ErrInternal.Message)
}

// Response 是统一响应体: data 仅在成功时输出, 失败时忽略该字段.
type Response struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (r Response) MarshalJSON() ([]byte, error) {
	type wire struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	}

	out := wire{Code: r.Code, Message: r.Message}
	if r.Code == CodeSuccess {
		out.Data = r.Data
		if out.Data == nil {
			out.Data = struct{}{}
		}
	}
	return json.Marshal(out)
}

// newFailure 统一映射: 未注册(code 0)的错误整体归入内部错误, 保证 code 与 message 一致.
func newFailure(def errs.Definition, message string) Response {
	if def.Code == 0 {
		def, message = errs.ErrInternal, errs.ErrInternal.Message
	}
	if def.Category == errs.CategoryBusiness {
		return Response{Code: int32(def.Code), Message: message}
	}
	return Response{Code: -int32(def.Code), Message: message}
}
