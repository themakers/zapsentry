package zapsentry

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func AttachCoreToLogger(sentryCore zapcore.Core, logger *zap.Logger) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	}))
}
