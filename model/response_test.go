package model

import (
	"encoding/json"
	"errors"
	"testing"

	errs "github.com/wensboy/quick_arch/internal/error"
)

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestSuccess_JSON(t *testing.T) {
	cases := []struct {
		name string
		data any
		want string
	}{
		{"with data", map[string]any{"id": 1}, `{"code":0,"message":"","data":{"id":1}}`},
		{"nil data", nil, `{"code":0,"message":"","data":{}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toJSON(t, Success(tc.data)); got != tc.want {
				t.Fatalf("json = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFailure_BusinessError(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"wrapped error", errs.New(errs.ErrNotFound)},
		{"bare definition", errs.ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toJSON(t, Failure(tc.err)); got != `{"code":20001,"message":"资源不存在"}` {
				t.Fatalf("json = %s", got)
			}
		})
	}
}

func TestFailure_InternalError(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"registered internal", errs.New(errs.ErrInternal)},
		{"bare internal definition", errs.ErrInternal},
		{"foreign error", errors.New("boom")},
		{"zero-code definition", errs.New(errs.ErrUnknown)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := Failure(tc.err)
			if resp.Code != -10000 {
				t.Fatalf("code = %d, want -10000", resp.Code)
			}
			if resp.Message != errs.ErrInternal.Message {
				t.Fatalf("message = %q, want %q", resp.Message, errs.ErrInternal.Message)
			}
		})
	}
}

func TestFailure_HidesInternalDetail(t *testing.T) {
	resp := Failure(errs.Wrapf(errs.ErrInternal, errors.New("dsn leaked"), "connect %s", "db"))
	if resp.Message != errs.ErrInternal.Message {
		t.Fatalf("message = %q, want default message", resp.Message)
	}
}

func TestFailure_NilIsSuccess(t *testing.T) {
	if got := Failure(nil); got.Code != CodeSuccess {
		t.Fatalf("code = %d, want %d", got.Code, CodeSuccess)
	}
}

func TestResponse_FailureDropsData(t *testing.T) {
	resp := Response{Code: -10000, Message: "内部错误", Data: map[string]any{"secret": 1}}
	if got := toJSON(t, resp); got != `{"code":-10000,"message":"内部错误"}` {
		t.Fatalf("json = %s, want data dropped", got)
	}
}
