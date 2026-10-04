package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/zgiai/luas/api/internal/infra/config"
)

func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buffer
}

func records(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()
	var result []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record), line)
		result = append(result, record)
	}
	return result
}

func TestStructuredLoggerWritesJSONForFailuresAndSlowQueries(t *testing.T) {
	buffer := captureSlog(t)
	log := newStructuredLogger(logger.Config{LogLevel: logger.Warn, SlowThreshold: 10 * time.Millisecond, IgnoreRecordNotFoundError: true})
	statement := func() (string, int64) { return `SELECT * FROM "users" WHERE id = $1`, 0 }
	ctx := context.Background()

	log.Trace(ctx, time.Now(), statement, errors.New("connection reset"))
	log.Trace(ctx, time.Now().Add(-time.Second), statement, nil)
	log.Trace(ctx, time.Now(), statement, gorm.ErrRecordNotFound)
	log.Trace(ctx, time.Now(), statement, nil)

	logged := records(t, buffer)
	require.Len(t, logged, 2, "not-found is ignored and fast queries stay quiet at warn level")
	assert.Equal(t, "database.query_failed", logged[0]["msg"])
	assert.Equal(t, "ERROR", logged[0]["level"])
	assert.Equal(t, "connection reset", logged[0]["err"])
	assert.Equal(t, `SELECT * FROM "users" WHERE id = $1`, logged[0]["sql"])
	assert.Equal(t, "database.slow_query", logged[1]["msg"])
	assert.Equal(t, "WARN", logged[1]["level"])
	assert.NotContains(t, buffer.String(), "\x1b[", "no terminal color codes")
}

func TestStructuredLoggerRespectsLevels(t *testing.T) {
	buffer := captureSlog(t)
	statement := func() (string, int64) { return "SELECT 1", 1 }
	newStructuredLogger(logger.Config{LogLevel: logger.Silent}).Trace(context.Background(), time.Now(), statement, errors.New("x"))
	assert.Empty(t, buffer.String())

	newStructuredLogger(logger.Config{}).LogMode(logger.Info).Trace(context.Background(), time.Now(), statement, nil)
	logged := records(t, buffer)
	require.Len(t, logged, 1)
	assert.Equal(t, "database.query", logged[0]["msg"])
	assert.Equal(t, "DEBUG", logged[0]["level"])
}

func TestGormLoggerFollowsJSONLogging(t *testing.T) {
	cfg := &config.Config{}
	cfg.Log.JSON = true
	_, structured := buildGormLogger(cfg).(*structuredLogger)
	assert.True(t, structured)

	cfg.Log.JSON = false
	_, structured = buildGormLogger(cfg).(*structuredLogger)
	assert.False(t, structured)
}
