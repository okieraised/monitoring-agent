package log

import (
	"fmt"
	"os"
	"sync"

	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/spf13/viper"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
}

var (
	defaultOnce   sync.Once
	defaultLogger *zap.Logger
	defaultErr    error
)

func logLevel() zap.AtomicLevel {
	switch viper.GetString(config.AgentLogLevel) {
	case "debug":
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		return zap.NewAtomicLevelAt(zap.FatalLevel)
	case "panic":
		return zap.NewAtomicLevelAt(zap.PanicLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}

// DefaultConfig returns a zap.Config configured with ECS-compatible encoders.
func DefaultConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = ecszap.ECSCompatibleEncoderConfig(cfg.EncoderConfig)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.Level = logLevel()
	return cfg
}

// InitDefault initializes the process-wide default logger once.
// Safe to call multiple times; only the first call wins.
func InitDefault(opts ...zap.Option) error {
	defaultOnce.Do(func() {
		cfg := DefaultConfig()
		defaultLogger, defaultErr = cfg.Build(opts...)
	})
	return defaultErr
}

// MustInitDefault is like InitDefault, but exits the process on failure.
func MustInitDefault(opts ...zap.Option) {
	if err := InitDefault(opts...); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize default logger: %v\n", err)
		os.Exit(1)
	}
}

// Default returns the default logger, initializing it if needed.
func Default() *Logger {
	if defaultLogger == nil {
		_ = InitDefault()
	}
	return &Logger{defaultLogger}
}

// Sync flushes any buffered logs on the default logger.
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}

// NewECSLogger builds a new, independent ECS-compatible logger.
func NewECSLogger(opts ...zap.Option) (*Logger, error) {
	cfg := DefaultConfig()
	l, err := cfg.Build(opts...)
	if err != nil {
		return nil, err
	}
	return &Logger{l}, nil
}

// MustNewECSLogger is NewECSLogger that panics on error.
func MustNewECSLogger(opts ...zap.Option) *Logger {
	l, err := NewECSLogger(opts...)
	if err != nil {
		panic(err)
	}
	return l
}

// With returns a child logger from this logger with the given fields.
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

// Named returns a child logger with the provided name.
func (l *Logger) Named(name string) *Logger {
	return &Logger{l.Logger.Named(name)}
}

// Sugar returns a sugared logger for this logger.
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.Logger.Sugar()
}

// WithString is a convenience for adding a single string field.
func (l *Logger) WithString(k, v string) *Logger {
	return l.With(zap.String(k, v))
}
