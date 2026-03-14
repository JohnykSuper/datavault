// Package logger wraps zap with a simplified structured-logging API.
// Never pass DEK, plaintext payload, or HMAC key as log fields.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a thin wrapper around zap.SugaredLogger.
type Logger struct {
	s *zap.SugaredLogger
}

// New constructs a Logger for the given level string (debug|info|warn|error).
func New(level string) *Logger {
	lvl := zapcore.InfoLevel
	_ = lvl.UnmarshalText([]byte(level))

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	z, _ := cfg.Build()
	return &Logger{s: z.Sugar()}
}

func (l *Logger) Info(msg string, kv ...interface{})  { l.s.Infow(msg, kv...) }
func (l *Logger) Debug(msg string, kv ...interface{}) { l.s.Debugw(msg, kv...) }
func (l *Logger) Warn(msg string, kv ...interface{})  { l.s.Warnw(msg, kv...) }
func (l *Logger) Error(msg string, kv ...interface{}) { l.s.Errorw(msg, kv...) }
func (l *Logger) Fatal(msg string, kv ...interface{}) { l.s.Fatalw(msg, kv...) }
func (l *Logger) With(kv ...interface{}) *Logger {
	return &Logger{s: l.s.With(kv...)}
}
