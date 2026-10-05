package log

import (
	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

const (
	KeyLevel      = "log.level"
	KeyEncoding   = "log.encoding"
	KeyOutput     = "log.output"
	KeyDir        = "log.dir"
	KeyFilename   = "log.filename"
	KeyMaxSize    = "log.maxSize"
	KeyMaxBackups = "log.maxBackups"
	KeyMaxAge     = "log.maxAge"
	KeyCompress   = "log.compress"
	KeyCaller     = "log.caller"
)

const (
	LevelDebug  = "debug"
	LevelInfo   = "info"
	LevelWarn   = "warn"
	LevelError  = "error"
	LevelDPanic = "dpanic"
	LevelPanic  = "panic"
	LevelFatal  = "fatal"
)

const (
	OutputStdout = "stdout"
	OutputStderr = "stderr"
	OutputFile   = "file"
	OutputBoth   = "both"
)

const (
	EncodingConsole = "console"
	EncodingJSON    = "json"
)

const (
	DefaultDir        = "data/log"
	DefaultFilename   = "app.log"
	DefaultMaxSize    = 100
	DefaultMaxAge     = 30
	DefaultMaxBackups = 7
)

type Config struct {
	Level      string
	Encoding   string
	Output     string
	Dir        string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Caller     bool
}

func DefaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Encoding:   EncodingConsole,
		Output:     OutputStdout,
		Dir:        DefaultDir,
		Filename:   DefaultFilename,
		MaxSize:    DefaultMaxSize,
		MaxBackups: DefaultMaxBackups,
		MaxAge:     DefaultMaxAge,
	}
}

func Load(cfg context2.ConfigContext) Config {
	c := DefaultConfig()
	if cfg == nil {
		return c
	}

	c.Level = config.String(cfg, KeyLevel, c.Level)
	c.Encoding = config.String(cfg, KeyEncoding, c.Encoding)
	c.Output = config.String(cfg, KeyOutput, c.Output)
	c.Dir = config.String(cfg, KeyDir, c.Dir)
	c.Filename = config.String(cfg, KeyFilename, c.Filename)
	c.MaxSize = config.Int(cfg, KeyMaxSize, c.MaxSize)
	c.MaxBackups = config.Int(cfg, KeyMaxBackups, c.MaxBackups)
	c.MaxAge = config.Int(cfg, KeyMaxAge, c.MaxAge)
	c.Compress = config.Bool(cfg, KeyCompress, c.Compress)
	c.Caller = config.Bool(cfg, KeyCaller, c.Caller)
	return c
}
