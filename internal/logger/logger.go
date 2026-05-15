// Package logger provides a structured JSON logger built on Go's slog package.
// JSON output is SIEM-compatible (Splunk, ELK, Datadog).
package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const reqIDKey contextKey = "request_id"

// Init initialises the global structured JSON logger.
func Init(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "ts" // shorter field name
			}
			return a
		},
	})
	slog.SetDefault(slog.New(h))
}

// WithRequestID attaches a request ID to a context for log correlation.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey, id)
}

// FromContext returns a logger enriched with the request ID from context.
func FromContext(ctx context.Context) *slog.Logger {
	l := slog.Default()
	if id, ok := ctx.Value(reqIDKey).(string); ok && id != "" {
		l = l.With("request_id", id)
	}
	return l
}

// AuditEvent emits a structured audit log entry.
// Fields: event, username, status, plus any extra slog.Attr pairs.
func AuditEvent(ctx context.Context, event, username, status string, extra ...slog.Attr) {
	args := []any{
		slog.String("event", event),
		slog.String("username", username),
		slog.String("status", status),
	}
	for _, a := range extra {
		args = append(args, a)
	}
	FromContext(ctx).Info("audit", args...)
}
