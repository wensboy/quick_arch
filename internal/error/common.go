package error

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type Category uint8

const (
	CategoryUnknown Category = iota
	CategorySystem
	CategoryBusiness
	CategoryExternal
)

func (c Category) String() string {
	switch c {
	case CategorySystem:
		return "system"
	case CategoryBusiness:
		return "business"
	case CategoryExternal:
		return "external"
	default:
		return "unknown"
	}
}

type Code uint32

const codeUnknown Code = 0

// Definition 是错误的标准定义 (code + 分类 + 默认消息).
// 实现 error, 可直接作为 errors.Is 的 target.
type Definition struct {
	Code     Code
	Category Category
	Message  string
}

func (d Definition) Error() string { return d.Message }

var (
	defsMu sync.RWMutex
	defs   = make(map[Code]Definition)
)

// Register 声明标准错误, 重复 code 会 panic.
func Register(code Code, category Category, message string) Definition {
	d := Definition{Code: code, Category: category, Message: message}

	defsMu.Lock()
	defer defsMu.Unlock()
	if _, ok := defs[code]; ok {
		panic(fmt.Sprintf("error: duplicate code %d", code))
	}
	defs[code] = d
	return d
}

func Lookup(code Code) (Definition, bool) {
	defsMu.RLock()
	defer defsMu.RUnlock()
	d, ok := defs[code]
	return d, ok
}

var (
	ErrUnknown  = Register(codeUnknown, CategoryUnknown, "未知错误")
	ErrInternal = Register(10000, CategorySystem, "内部错误")
	ErrTimeout  = Register(10001, CategorySystem, "处理超时")
	ErrExternal = Register(10002, CategoryExternal, "外部依赖错误")
)

var (
	ErrInvalidParam = Register(20000, CategoryBusiness, "参数非法")
	ErrNotFound     = Register(20001, CategoryBusiness, "资源不存在")
	ErrConflict     = Register(20002, CategoryBusiness, "资源冲突")
	ErrPermission   = Register(20003, CategoryBusiness, "无权限")
)

// Error 是带标准化定义, 调用位置与上下文字段的错误实例.
type Error struct {
	def    Definition
	msg    string
	cause  error
	fields map[string]any
	at     string
}

func New(def Definition) *Error {
	return newError(def, def.Message, nil)
}

func Newf(def Definition, format string, args ...any) *Error {
	return newError(def, fmt.Sprintf(format, args...), nil)
}

func Wrap(def Definition, cause error) *Error {
	return newError(def, def.Message, cause)
}

func Wrapf(def Definition, cause error, format string, args ...any) *Error {
	return newError(def, fmt.Sprintf(format, args...), cause)
}

func newError(def Definition, msg string, cause error) *Error {
	return &Error{def: def, msg: msg, cause: cause, at: caller()}
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%d] %s", e.def.Code, e.msg)
	if e.cause != nil {
		b.WriteString(": " + e.cause.Error())
	}
	if len(e.fields) > 0 {
		b.WriteString(" " + formatFields(e.fields))
	}
	if e.at != "" {
		b.WriteString(" @ " + e.at)
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.cause }

// Is 支持 errors.Is(err, ErrNotFound) 与 errors.Is(err, otherErr) 按 code 匹配.
func (e *Error) Is(target error) bool {
	switch t := target.(type) {
	case Definition:
		return e.def.Code == t.Code
	case *Error:
		return e.def.Code == t.def.Code
	default:
		return false
	}
}

func (e *Error) Definition() Definition { return e.def }
func (e *Error) Code() Code             { return e.def.Code }
func (e *Error) Category() Category     { return e.def.Category }
func (e *Error) Message() string        { return e.msg }
func (e *Error) Cause() error           { return e.cause }
func (e *Error) Caller() string         { return e.at }
func (e *Error) Fields() map[string]any { return e.fields }

func (e *Error) With(kv ...any) *Error {
	for i := 0; i+1 < len(kv); i += 2 {
		e.WithField(fmt.Sprint(kv[i]), kv[i+1])
	}
	return e
}

func (e *Error) WithField(key string, value any) *Error {
	if e.fields == nil {
		e.fields = make(map[string]any, 1)
	}
	e.fields[key] = value
	return e
}

// FromError 从错误链中提取 *Error.
func FromError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// CodeOf 返回错误链最外层标准化错误的 code, 非本包错误返回 codeUnknown.
func CodeOf(err error) Code {
	if e, ok := FromError(err); ok {
		return e.def.Code
	}
	return codeUnknown
}

// CategoryOf 返回错误链最外层标准化错误的分类, 非本包错误返回 CategoryUnknown.
func CategoryOf(err error) Category {
	if e, ok := FromError(err); ok {
		return e.def.Category
	}
	return CategoryUnknown
}

// IsCode 判断错误链中是否存在指定 code.
func IsCode(err error, code Code) bool {
	d, ok := Lookup(code)
	if !ok {
		return false
	}
	return errors.Is(err, d)
}

// IsCategory 判断错误链中是否存在指定分类.
func IsCategory(err error, category Category) bool {
	for err != nil {
		if e, ok := err.(*Error); ok && e.def.Category == category {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

func formatFields(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

var (
	pkgDir         = pkgDirOfThisFile()
	internalFrames = map[string]bool{"caller": true, "newError": true, "New": true, "Newf": true, "Wrap": true, "Wrapf": true}
)

func pkgDirOfThisFile() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func caller() string {
	pcs := make([]uintptr, 16)
	n := runtime.Callers(2, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if !isInternalFrame(frame) {
			return fmt.Sprintf("%s:%d", shortFile(frame.File), frame.Line)
		}
		if !more {
			return ""
		}
	}
}

func isInternalFrame(frame runtime.Frame) bool {
	if filepath.Dir(frame.File) != pkgDir {
		return false
	}
	name := frame.Function
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return internalFrames[name]
}

func shortFile(file string) string {
	parts := strings.Split(filepath.ToSlash(file), "/")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, "/")
}
