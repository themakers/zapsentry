package zapsentry

import (
	"errors"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

var ErrFlushTimedOut = errors.New("flush timed out")

func NewCore(cfg Config, factory SentryClientFactory) (zapcore.Core, error) {
	client, err := factory()
	if err != nil {
		return nil, err
	}

	core := core{
		client:       client,
		cfg:          &cfg,
		LevelEnabler: cfg.Level,
		fields:       make(map[string]interface{}),
	}

	if core.cfg.FlushTimeout <= 0 {
		core.cfg.FlushTimeout = 5 * time.Second
	}

	return &core, nil
}

var _ zapcore.Core = (*core)(nil)

type core struct {
	client *sentry.Client
	cfg    *Config
	zapcore.LevelEnabler

	fields map[string]interface{}
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.Level.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	event := sentry.NewEvent()
	event.Message = ent.Message
	event.Timestamp = ent.Time
	event.Level = sentrySeverity(ent.Level)
	event.Platform = "Golang"
	event.Extra = clone.fields
	event.Tags = c.cfg.Tags

	if !c.cfg.DisableStacktrace {
		trace := sentry.NewStacktrace()
		trace.Frames = trace.Frames[4:]
		if trace != nil {
			event.Exception = []sentry.Exception{{
				Type:       ent.Message,
				Value:      ent.Caller.TrimmedPath(),
				Stacktrace: trace,
			}}
		}
	}

	hub := c.cfg.Hub
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	c.client.CaptureEvent(event, nil, hub.Scope())

	return nil
}

func newStacktrace() *sentry.Stacktrace {
	pcs := make([]uintptr, 100)
	n := runtime.Callers(1, pcs)

	if n == 0 {
		return nil
	}

	frames := extractFrames(pcs[:n])
	frames = filterFrames(frames)

	stacktrace := Stacktrace{
		Frames: frames,
	}

	return &stacktrace
}

func (c *core) Sync() error {
	if !c.client.Flush(c.cfg.FlushTimeout) {
		return ErrFlushTimedOut
	} else {
		return nil
	}
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		fields:       m,
		LevelEnabler: c.LevelEnabler,
	}
}

type ClientGetter interface {
	GetClient() *sentry.Client
}

func (c *core) GetClient() *sentry.Client {
	return c.client
}
