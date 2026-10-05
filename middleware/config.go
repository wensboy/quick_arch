package middleware

import (
	"strings"

	"github.com/wensboy/quick_arch/internal/config"
)

func RegisterConfig(r *config.Registry) {
	for _, name := range filterable {
		env := "MIDDLEWARE_" + strings.ToUpper(name)
		r.Register(
			config.Entry{Key: keyInclude(name), Type: config.FlagTypeString, Usage: name + " 生效规则前缀 (rest 为 URI 路径, rpc 为函数名), 逗号分隔", Env: env + "_INCLUDE"},
			config.Entry{Key: keyExclude(name), Type: config.FlagTypeString, Usage: name + " 放行规则前缀 (rest 为 URI 路径, rpc 为函数名), 逗号分隔", Env: env + "_EXCLUDE"},
		)
	}
}

// filterable 是需要按路由放行控制的中间件.
var filterable = []string{NameAccessLog, NameDebug}

func keyInclude(name string) string { return "middleware." + name + ".include" }
func keyExclude(name string) string { return "middleware." + name + ".exclude" }
