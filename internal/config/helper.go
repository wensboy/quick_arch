package config

import (
	"fmt"
	"strconv"
	"strings"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

// FlagType* 是 Entry.Type 的唯一类型词汇, 供各模块注册与 flag 构建共用.
const (
	FlagTypeString = "string"
	FlagTypeInt    = "int"
	FlagTypeFloat  = "float"
	FlagTypeBool   = "bool"
)

func String(cfg context2.ConfigContext, key, def string) string {
	v, ok := lookup(cfg, key)
	if !ok {
		return def
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func Int(cfg context2.ConfigContext, key string, def int) int {
	v, ok := lookup(cfg, key)
	if !ok {
		return def
	}
	return toInt(v, def)
}

func Bool(cfg context2.ConfigContext, key string, def bool) bool {
	v, ok := lookup(cfg, key)
	if !ok {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(t))
		if err != nil {
			return def
		}
		return b
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return def
	}
}

// List 兼容 []string / []any / 逗号分隔字符串三种来源.
func List(cfg context2.ConfigContext, key string, def []string) []string {
	v, ok := lookup(cfg, key)
	if !ok {
		return def
	}
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case string:
		return SplitList(t)
	default:
		return def
	}
}

// Maps 读取"对象列表"型配置, 兼容 []map[string]any 与 JSON 解析出的 []any.
func Maps(cfg context2.ConfigContext, key string) []map[string]any {
	v, ok := lookup(cfg, key)
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func SplitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return def
		}
		return n
	default:
		return def
	}
}

func lookup(cfg context2.ConfigContext, key string) (any, bool) {
	if cfg == nil {
		return nil, false
	}
	v, ok := cfg.Lookup(key)
	if !ok || v == nil {
		return nil, false
	}
	return v, true
}
