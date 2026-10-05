package log

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
)

var _ context2.LogContext = (*Logger)(nil)

type Logger struct {
	sugar  *zap.SugaredLogger
	closer io.Closer
}

func New(cfg context2.ConfigContext) (*Logger, error) {
	return NewWithConfig(Load(cfg))
}

func NewWithConfig(c Config) (*Logger, error) {
	level, err := parseLevel(c.Level)
	if err != nil {
		return nil, err
	}

	ws, closer, err := buildWriteSyncer(c)
	if err != nil {
		return nil, err
	}

	var encoder zapcore.Encoder
	if strings.EqualFold(strings.TrimSpace(c.Encoding), EncodingJSON) {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	} else {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}

	opts := make([]zap.Option, 0, 1)
	if c.Caller {
		opts = append(opts, zap.AddCaller())
	}

	return &Logger{
		sugar:  zap.New(zapcore.NewCore(encoder, ws, level), opts...).Sugar(),
		closer: closer,
	}, nil
}

func (l *Logger) Log(level string, msg string, kv ...any) {
	l.sugar.Logw(l.resolveLevel(level), msg, kv...)
}

func (l *Logger) Logf(level string, format string, args ...any) {
	l.sugar.Logf(l.resolveLevel(level), format, args...)
}

func (l *Logger) Sync() error {
	return l.sugar.Sync()
}

func (l *Logger) Close() error {
	err := l.Sync()
	if l.closer != nil {
		if cerr := l.closer.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

// resolveLevel 非法级别回退到 info.
func (l *Logger) resolveLevel(level string) zapcore.Level {
	lv, err := parseLevel(level)
	if err != nil {
		return zapcore.InfoLevel
	}
	return lv
}

func parseLevel(level string) (zapcore.Level, error) {
	level = strings.TrimSpace(level)
	if level == "" {
		return zapcore.InfoLevel, nil
	}
	var lv zapcore.Level
	if err := lv.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		return 0, errs.New(ErrInvalidLevel).With("level", level)
	}
	return lv, nil
}

func buildWriteSyncer(c Config) (zapcore.WriteSyncer, io.Closer, error) {
	switch strings.ToLower(strings.TrimSpace(c.Output)) {
	case OutputStderr:
		return zapcore.AddSync(os.Stderr), nil, nil

	case OutputFile, OutputBoth:
		rotator := &lumberjack.Logger{
			Filename:   filepath.Join(dirOrDot(c.Dir), filenameOrDefault(c.Filename)),
			MaxSize:    positiveOrDefault(c.MaxSize, DefaultMaxSize),
			MaxBackups: c.MaxBackups,
			MaxAge:     c.MaxAge,
			Compress:   c.Compress,
		}
		if strings.EqualFold(strings.TrimSpace(c.Output), OutputBoth) {
			return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(rotator)), rotator, nil
		}
		return zapcore.AddSync(rotator), rotator, nil

	case OutputStdout, "":
		return zapcore.AddSync(os.Stdout), nil, nil

	default:
		return nil, nil, errs.New(ErrInvalidOutput).With("output", c.Output)
	}
}

func dirOrDot(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return "."
	}
	return dir
}

func filenameOrDefault(name string) string {
	if strings.TrimSpace(name) == "" {
		return DefaultFilename
	}
	return name
}

func positiveOrDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
