package log

import "github.com/wensboy/quick_arch/internal/config"

func RegisterConfig(r *config.Registry) {
	c := DefaultConfig()
	r.Register(
		config.Entry{Key: KeyLevel, Flag: "log-level", Type: config.FlagTypeString, Usage: "日志级别: debug|info|warn|error", Env: "LOG_LEVEL", Default: c.Level},
		config.Entry{Key: KeyEncoding, Flag: "log-encoding", Type: config.FlagTypeString, Usage: "日志编码: console|json", Env: "LOG_ENCODING", Default: c.Encoding},
		config.Entry{Key: KeyOutput, Flag: "log-output", Type: config.FlagTypeString, Usage: "日志输出: stdout|stderr|file|both", Env: "LOG_OUTPUT", Default: c.Output},
		config.Entry{Key: KeyDir, Flag: "log-dir", Type: config.FlagTypeString, Usage: "日志目录", Env: "LOG_DIR", Default: c.Dir},
		config.Entry{Key: KeyFilename, Flag: "log-filename", Type: config.FlagTypeString, Usage: "日志文件名", Env: "LOG_FILENAME", Default: c.Filename},
		config.Entry{Key: KeyMaxSize, Flag: "log-max-size", Type: config.FlagTypeInt, Usage: "单文件大小上限(MB)", Env: "LOG_MAX_SIZE", Default: c.MaxSize},
		config.Entry{Key: KeyMaxBackups, Flag: "log-max-backups", Type: config.FlagTypeInt, Usage: "保留的历史文件数", Env: "LOG_MAX_BACKUPS", Default: c.MaxBackups},
		config.Entry{Key: KeyMaxAge, Flag: "log-max-age", Type: config.FlagTypeInt, Usage: "历史文件保留天数", Env: "LOG_MAX_AGE", Default: c.MaxAge},
		config.Entry{Key: KeyCompress, Flag: "log-compress", Type: config.FlagTypeBool, Usage: "压缩历史日志文件", Env: "LOG_COMPRESS", Default: c.Compress},
		config.Entry{Key: KeyCaller, Flag: "log-caller", Type: config.FlagTypeBool, Usage: "记录调用位置", Env: "LOG_CALLER", Default: c.Caller},
	)
}
