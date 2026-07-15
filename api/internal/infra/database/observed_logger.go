package database

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/exception"
)

type observedLogger struct {
	base logger.Interface
}

var _ gorm.ParamsFilter = (*observedLogger)(nil)

func init() {
	// GORM's Scan path records SQL through a package-level trace recorder before
	// handing it back to the configured logger. Keep that path parameterized too.
	logger.RecorderParamsFilter = parameterizedQueryFilter
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

// ParamsFilter keeps GORM from interpolating bound values before Trace sees SQL.
// The wrapper must implement this optional GORM interface itself; otherwise the
// wrapped logger's ParameterizedQueries setting is bypassed.
func (l *observedLogger) ParamsFilter(
	ctx context.Context,
	sql string,
	params ...interface{},
) (string, []interface{}) {
	return parameterizedQueryFilter(ctx, sql, params...)
}

func parameterizedQueryFilter(
	_ context.Context,
	sql string,
	_ ...interface{},
) (string, []interface{}) {
	return sql, nil
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
