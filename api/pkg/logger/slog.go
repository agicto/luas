package logger

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// SlogHandler routes log/slog records into a platform Logger, so code that logs through the
// standard library shares the configured console, JSON, file, and Sentry outputs.
type SlogHandler struct {
	logger *Logger
	// fields holds WithAttrs values already flattened under the groups open when they were added.
	fields map[string]any
	groups []string
}

// NewSlogHandler returns a slog.Handler backed by the given Logger. Install it with
// slog.SetDefault(slog.New(logger.NewSlogHandler(l))).
func NewSlogHandler(l *Logger) *SlogHandler {
	return &SlogHandler{logger: l}
}

// Enabled defers level filtering to the Logger's own handlers.
func (h *SlogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *SlogHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := make(map[string]any, len(h.fields)+record.NumAttrs())
	for key, value := range h.fields {
		fields[key] = value
	}
	prefix := h.prefix()
	record.Attrs(func(attr slog.Attr) bool {
		addAttr(fields, prefix, attr)
		return true
	})
	h.logger.log(ctx, slogLevel(record.Level), record.Message, fields)
	return nil
}

func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copied := *h
	copied.fields = make(map[string]any, len(h.fields)+len(attrs))
	for key, value := range h.fields {
		copied.fields[key] = value
	}
	prefix := h.prefix()
	for _, attr := range attrs {
		addAttr(copied.fields, prefix, attr)
	}
	return &copied
}

func (h *SlogHandler) prefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	copied := *h
	copied.groups = append(append([]string(nil), h.groups...), name)
	return &copied
}

func slogLevel(level slog.Level) Level {
	switch {
	case level >= slog.LevelError:
		return LevelError
	case level >= slog.LevelWarn:
		return LevelWarning
	case level >= slog.LevelInfo:
		return LevelInfo
	default:
		return LevelDebug
	}
}

func addAttr(fields map[string]any, prefix string, attr slog.Attr) {
	value := attr.Value.Resolve()
	if attr.Key == "" && value.Kind() != slog.KindGroup {
		return
	}
	if value.Kind() == slog.KindGroup {
		groupPrefix := prefix
		if attr.Key != "" {
			groupPrefix = prefix + attr.Key + "."
		}
		for _, member := range value.Group() {
			addAttr(fields, groupPrefix, member)
		}
		return
	}
	fields[prefix+attr.Key] = slogValue(value)
}

func slogValue(value slog.Value) any {
	switch value.Kind() {
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindAny:
		if err, ok := value.Any().(error); ok {
			return err.Error()
		}
		return value.Any()
	default:
		return value.Any()
	}
}
