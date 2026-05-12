package exception

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	zerrors "github.com/zgiai/zgo/pkg/errors"
	"github.com/zgiai/zgo/pkg/logger"
	"github.com/zgiai/zgo/pkg/response"
	"go.opentelemetry.io/otel/trace"
)

// Recovery creates a development-friendly exception center middleware.
// In debug mode it renders an HTML exception page for browser requests while
// preserving the normal JSON error contract for API clients.
func Recovery(debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		collector := NewCollector(c.Request)
		c.Request = c.Request.WithContext(WithCollector(c.Request.Context(), collector))

		defer func() {
			if r := recover(); r != nil {
				appErr := panicToError(r)
				logPanic(c, appErr)

				if debug && wantsHTML(c) {
					data := buildDebugPageData(c, appErr, collector)
					zerrors.RenderDebugPageData(c, appErr.Status, data)
					c.Abort()
					return
				}

				response.Error(c, http.StatusInternalServerError, "Internal server error")
				c.Abort()
			}
		}()

		c.Next()
	}
}

func panicToError(value any) *zerrors.AppError {
	err := zerrors.LegacyInternal("A panic occurred")
	switch v := value.(type) {
	case *zerrors.AppError:
		err = v
	case error:
		err = zerrors.LegacyInternal("A panic occurred").WithInternal(v)
	case string:
		err = zerrors.LegacyInternal(v)
	default:
		err = zerrors.LegacyInternal(fmt.Sprintf("Unknown panic: %v", v))
	}

	// Grow the buffer until runtime.Stack fits the whole trace. 8KB was
	// silently truncating deep traces (gorm/gin pipelines easily hit 20KB+).
	stack := make([]byte, 8192)
	for {
		n := runtime.Stack(stack, false)
		if n < len(stack) {
			err.Stack = string(stack[:n])
			break
		}
		if len(stack) >= 1<<20 { // 1MB safety ceiling
			err.Stack = string(stack[:n])
			break
		}
		stack = make([]byte, 2*len(stack))
	}

	frames := zerrors.DebugStackFrames(err.Stack)
	if len(frames) > 0 {
		err.File = frames[0].File
		err.Line = frames[0].Line
	}

	return err
}

func logPanic(c *gin.Context, err *zerrors.AppError) {
	fields := map[string]any{
		"error_code": err.Code,
		"request_id": c.GetString("request_id"),
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"route_name": c.GetString("route_name"),
	}
	if traceID, ok := traceIDs(c.Request.Context()); ok {
		fields["trace_id"] = traceID
	}
	logger.Error("panic recovered", fields)
}

func wantsHTML(c *gin.Context) bool {
	accept := strings.ToLower(strings.TrimSpace(c.GetHeader("Accept")))
	if accept == "" {
		return true
	}
	if strings.Contains(accept, "application/json") {
		return false
	}
	return strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}

func buildDebugPageData(c *gin.Context, err *zerrors.AppError, collector *Collector) zerrors.DebugPageData {
	requestID := c.GetString("request_id")
	traceID, hasTrace := traceIDs(c.Request.Context())

	data := zerrors.DebugPageData{
		Title:      string(err.Code),
		Message:    err.Message,
		Code:       string(err.Code),
		Status:     err.Status,
		File:       err.File,
		Line:       err.Line,
		Stack:      zerrors.DebugStackFrames(err.Stack),
		RouteName:  c.GetString("route_name"),
		RequestID:  requestID,
		TraceID:    traceID,
		Request:    zerrors.RequestInfo{Method: c.Request.Method, URL: c.Request.URL.String()},
		SQLQueries: buildSQLDebugRows(collector),
		RecentLogs: buildRecentLogRows(requestID, traceID),
		Environment: map[string]string{
			"Go Version": runtime.Version(),
			"OS":         runtime.GOOS,
			"Arch":       runtime.GOARCH,
		},
	}
	if !hasTrace {
		data.TraceID = ""
	}

	if collector != nil {
		if method := collector.Method(); method != "" {
			data.Request.Method = method
		}
		if url := collector.URL(); url != "" {
			data.Request.URL = url
		}
		data.Request.Headers = collector.Headers()
		data.Request.Query = collector.Query()
	}

	return data
}

func buildSQLDebugRows(collector *Collector) []zerrors.DebugSQLQuery {
	if collector == nil {
		return nil
	}

	queries := collector.SQL()
	rows := make([]zerrors.DebugSQLQuery, 0, len(queries))
	for _, query := range queries {
		rows = append(rows, zerrors.DebugSQLQuery{
			Time:         query.Time.Format("15:04:05.000"),
			Duration:     query.Duration.String(),
			Statement:    query.Statement,
			RowsAffected: query.RowsAffected,
			Error:        query.Error,
		})
	}
	return rows
}

func buildRecentLogRows(requestID, traceID string) []zerrors.DebugLogEntry {
	entries := logger.DefaultMemoryHandler().RecentByRequest(requestID, traceID, 12)
	if len(entries) == 0 {
		entries = logger.DefaultMemoryHandler().Recent(8)
	}

	rows := make([]zerrors.DebugLogEntry, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, zerrors.DebugLogEntry{
			Time:      entry.Time.Format("15:04:05.000"),
			Level:     entry.Level.String(),
			Channel:   entry.Channel,
			Message:   entry.Message,
			RequestID: entry.RequestID,
			TraceID:   entry.TraceID,
			Context:   prettyContext(entry.Context),
		})
	}
	return rows
}

func prettyContext(ctx map[string]any) string {
	if len(ctx) == 0 {
		return ""
	}
	payload, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return ""
	}
	return string(payload)
}

func traceIDs(ctx context.Context) (string, bool) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return "", false
	}
	return span.SpanContext().TraceID().String(), true
}
