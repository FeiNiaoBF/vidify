package logging

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger exposes the logging contract consumed by higher layers.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error
}

// Field represents a key/value pair attached to log entries.
type Field struct {
	Key   string
	Value any
}

// Factory constructs loggers according to configuration.
type Factory struct{}

// NewFactory returns a Factory instance.
func NewFactory() Factory {
	return Factory{}
}

// Build instantiates a logger based on the provided settings.
func (Factory) Build(cfg Config) (Logger, error) {
	zapCfg := zap.NewProductionConfig()
	if strings.EqualFold(cfg.Mode, "development") {
		zapCfg = zap.NewDevelopmentConfig()
	}

	if cfg.Level != "" {
		var lvl zapcore.Level
		if err := lvl.Set(strings.ToLower(cfg.Level)); err != nil {
			return nil, fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
		}
		zapCfg.Level = zap.NewAtomicLevelAt(lvl)
	}

	coreLogger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}

	return &zapLogger{logger: coreLogger.Sugar()}, nil
}

// Config mirrors configuration.LoggingConfig but duplicated to decouple packages.
type Config struct {
	Level string
	Mode  string
}

type zapLogger struct {
	logger *zap.SugaredLogger
}

func (l *zapLogger) Debug(msg string, fields ...Field) { l.logger.Debugw(msg, flatten(fields)...) }
func (l *zapLogger) Info(msg string, fields ...Field)  { l.logger.Infow(msg, flatten(fields)...) }
func (l *zapLogger) Warn(msg string, fields ...Field)  { l.logger.Warnw(msg, flatten(fields)...) }
func (l *zapLogger) Error(msg string, fields ...Field) { l.logger.Errorw(msg, flatten(fields)...) }

func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{logger: l.logger.With(flatten(fields)...)}
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

func flatten(fields []Field) []any {
	if len(fields) == 0 {
		return nil
	}
	out := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		out = append(out, field.Key, field.Value)
	}
	return out
}
