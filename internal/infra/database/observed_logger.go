package database

import (
	"context"
	"time"

	"github.com/zgiai/zgo/internal/infra/exception"
	"gorm.io/gorm/logger"
)

type observedLogger struct {
	base logger.Interface
}

func wrapObservedLogger(base logger.Interface) logger.Interface {
	if base == nil {
		return nil
	}
	return &observedLogger{base: base}
}

func (l *observedLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &observedLogger{base: l.base.LogMode(level)}
}

func (l *observedLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.base.Info(ctx, msg, data...)
}

func (l *observedLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.base.Warn(ctx, msg, data...)
}

func (l *observedLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.base.Error(ctx, msg, data...)
}

func (l *observedLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if collector := exception.FromContext(ctx); collector != nil {
		statement, rowsAffected := fc()
		collector.AddSQL(begin, time.Since(begin), statement, rowsAffected, err)
		l.base.Trace(ctx, begin, func() (string, int64) {
			return statement, rowsAffected
		}, err)
		return
	}

	l.base.Trace(ctx, begin, fc, err)
}
