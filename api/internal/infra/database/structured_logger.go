package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// structuredLogger sends GORM output through slog, so database errors and slow queries follow the
// application's JSON logging instead of GORM's colored text. Statements are already parameterized
// by the observed logger wrapper; bound values never reach the log.
type structuredLogger struct {
	level          logger.LogLevel
	slowThreshold  time.Duration
	ignoreNotFound bool
}

func newStructuredLogger(config logger.Config) logger.Interface {
	return &structuredLogger{
		level:          config.LogLevel,
		slowThreshold:  config.SlowThreshold,
		ignoreNotFound: config.IgnoreRecordNotFoundError,
	}
}

func (l *structuredLogger) LogMode(level logger.LogLevel) logger.Interface {
	copied := *l
	copied.level = level
	return &copied
}

func (l *structuredLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		slog.InfoContext(ctx, "database.message", "message", fmt.Sprintf(msg, data...))
	}
}

func (l *structuredLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		slog.WarnContext(ctx, "database.message", "message", fmt.Sprintf(msg, data...))
	}
}

func (l *structuredLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		slog.ErrorContext(ctx, "database.message", "message", fmt.Sprintf(msg, data...))
	}
}

func (l *structuredLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	ignored := l.ignoreNotFound && errors.Is(err, gorm.ErrRecordNotFound)
	switch {
	case err != nil && l.level >= logger.Error && !ignored:
		statement, rows := fc()
		slog.ErrorContext(ctx, "database.query_failed",
			"err", err, "sql", statement, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= logger.Warn:
		statement, rows := fc()
		slog.WarnContext(ctx, "database.slow_query",
			"sql", statement, "rows", rows, "elapsed_ms", elapsed.Milliseconds(),
			"threshold_ms", l.slowThreshold.Milliseconds())
	case l.level >= logger.Info:
		statement, rows := fc()
		slog.DebugContext(ctx, "database.query", "sql", statement, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	}
}
