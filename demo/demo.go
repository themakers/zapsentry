package main

import (
	"errors"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/themakers/zapsentry"
	"go.uber.org/zap"
)

var CommitSHA string

// SENTRY_DSN="$YOUR_DSN" go run -ldflags "-X main.CommitSHA=$(git rev-list -1 HEAD)" demo.go

func modifyToSentryLogger(logger *zap.Logger, dsn string) (func(), *zap.Logger) {
	cfg := zapsentry.Config{
		//Level:        zapcore.ErrorLevel,
		FlushTimeout: 3 * time.Second,
		Tags: map[string]string{
			"component": "master",
		},
	}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              dsn,
		Transport:        &sentry.HTTPSyncTransport{},
		AttachStacktrace: true,
		Release:          CommitSHA,
	})
	if err != nil {
		panic(err)
	}

	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))
	if err != nil {
		panic(err)
	}

	return func() {
		if err := core.Sync(); err != nil {
			panic(err)
		}
	}, zapsentry.AttachCoreToLogger(core, logger)
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	sync, logger := modifyToSentryLogger(logger, "")
	defer sync()

	logger.Error("test error",
		zap.String("sample field", "sample field value"),
		zap.Error(errors.New("sample description")))

	Deeper(logger)
}

func Deeper(logger *zap.Logger) {
	logger.Error("test error 2",
		zap.String("sample field", "sample field value"),
		zap.Error(errors.New("sample description")))

	logger.Panic("panic attack")
}
