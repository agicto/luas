package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Config holds error handler configuration
type Config struct {
	// Debug mode shows detailed error information
	Debug bool

	// ShowStack shows stack traces in debug mode
	ShowStack bool

	// LogErrors logs errors to console
	LogErrors bool

	// Custom error response transformer
	Transformer func(*gin.Context, *AppError) interface{}

	// Custom error logger
	Logger func(*gin.Context, *AppError)

	// Custom recovery handler
	RecoveryHandler func(*gin.Context, interface{})
}

// DefaultConfig returns default error handler config
func DefaultConfig() Config {
	mode := os.Getenv("SERVER_MODE")
	if mode == "" {
		mode = os.Getenv("GIN_MODE")
	}
	return Config{
		Debug:     mode != "release",
		ShowStack: mode != "release",
		LogErrors: true,
	}
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     ErrorDetail            `json:"error"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    Code                `json:"code"`
	Message string              `json:"message"`
	Detail  string              `json:"detail,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
	Stack   []StackFrame        `json:"stack,omitempty"`
}

// StackFrame represents a stack frame for debugging
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// Handler returns a Gin error handling middleware
func Handler(cfg ...Config) gin.HandlerFunc {
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}

	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, err, config)
		}
	}
}

// Recovery returns a panic recovery middleware with pretty error output
func Recovery(cfg ...Config) gin.HandlerFunc {
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}

	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Custom recovery handler
				if config.RecoveryHandler != nil {
					config.RecoveryHandler(c, r)
					return
				}

				// Create internal error
				var err *AppError
				switch v := r.(type) {
				case *AppError:
					err = v
				case error:
					err = LegacyInternal("A panic occurred").WithInternal(v)
				case string:
					err = LegacyInternal(v)
				default:
					err = LegacyInternal(fmt.Sprintf("Unknown panic: %v", r))
				}

				// Capture stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				err.Stack = string(buf[:n])

				handleError(c, err, config)
			}
		}()

		c.Next()
	}
}

// handleError processes and responds with an error
func handleError(c *gin.Context, err error, config Config) {
	appErr := toAppError(err)

	// Log error
	if config.LogErrors {
		logError(c, appErr, config)
	}

	// Custom logger
	if config.Logger != nil {
		config.Logger(c, appErr)
	}

	// Build response
	var response interface{}
	if config.Transformer != nil {
		response = config.Transformer(c, appErr)
	} else {
		response = buildErrorResponse(c, appErr, config)
	}

	c.AbortWithStatusJSON(appErr.Status, response)
}

// toAppError converts any error to AppError
func toAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// Check for common error types
	switch e := err.(type) {
	case interface{ Status() int }:
		return &AppError{
			Status:   e.Status(),
			Code:     CodeUnknown,
			Message:  err.Error(),
			Internal: err,
		}
	default:
		return LegacyInternal(err.Error()).WithInternal(err)
	}
}

// buildErrorResponse builds the standard error response
func buildErrorResponse(c *gin.Context, err *AppError, config Config) ErrorResponse {
	detail := ErrorDetail{
		Code:    err.Code,
		Message: err.Message,
		Errors:  err.Errors,
	}

	// Add detail in debug mode
	if config.Debug && err.Detail != "" {
		detail.Detail = err.Detail
	}

	// Add stack trace in debug mode
	if config.Debug && config.ShowStack && err.Stack != "" {
		detail.Stack = parseStackTrace(err.Stack)
	}

	response := ErrorResponse{
		Success:   false,
		Error:     detail,
		Meta:      err.Meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Add request ID if available
	if requestID := c.GetString("request_id"); requestID != "" {
		response.RequestID = requestID
	}

	return response
}

// parseStackTrace parses a stack trace string into frames
func parseStackTrace(stack string) []StackFrame {
	var frames []StackFrame
	lines := strings.Split(stack, "\n")

	for i := 0; i < len(lines)-1; i += 2 {
		funcLine := strings.TrimSpace(lines[i])
		if i+1 >= len(lines) {
			break
		}
		fileLine := strings.TrimSpace(lines[i+1])

		// Skip runtime and gin internal frames
		if strings.Contains(funcLine, "runtime.") ||
			strings.Contains(funcLine, "gin-gonic") ||
			strings.Contains(funcLine, "pkg/errors") {
			continue
		}

		// Parse file:line
		file := fileLine
		line := 0
		if idx := strings.LastIndex(fileLine, ":"); idx != -1 {
			file = fileLine[:idx]
			fmt.Sscanf(fileLine[idx+1:], "%d", &line)
		}

		// Parse function name
		funcName := funcLine
		if idx := strings.LastIndex(funcLine, "/"); idx != -1 {
			funcName = funcLine[idx+1:]
		}

		frames = append(frames, StackFrame{
			File:     file,
			Line:     line,
			Function: funcName,
		})

		// Limit frames
		if len(frames) >= 10 {
			break
		}
	}

	return frames
}

// logError logs the error to console
func logError(c *gin.Context, err *AppError, config Config) {
	// Build log message
	msg := fmt.Sprintf("[ERROR] %s %s - %d %s: %s",
		c.Request.Method,
		c.Request.URL.Path,
		err.Status,
		err.Code,
		err.Message,
	)

	if err.Internal != nil {
		msg += fmt.Sprintf(" (internal: %v)", err.Internal)
	}

	fmt.Println(msg)

	// Print stack in debug mode
	if config.Debug && config.ShowStack && err.Stack != "" {
		fmt.Println("Stack trace:")
		fmt.Println(err.Stack)
	}
}

// --- Gin Integration Helpers ---

// Abort aborts with an error
func Abort(c *gin.Context, err *AppError) {
	c.Error(err)
	c.Abort()
}

// AbortWithValidation aborts with validation errors
func AbortWithValidation(c *gin.Context, errors map[string][]string) {
	Abort(c, LegacyValidationWithErrors(errors))
}

// AbortWithMessage aborts with a message
func AbortWithMessage(c *gin.Context, status int, message string) {
	err := &AppError{
		Status:  status,
		Code:    statusToCode(status),
		Message: message,
	}
	Abort(c, err)
}

// statusToCode converts HTTP status to error code
func statusToCode(status int) Code {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusUnprocessableEntity:
		return CodeValidation
	case http.StatusTooManyRequests:
		return CodeTooManyRequests
	case http.StatusServiceUnavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternal
	}
}

// --- Problem Details (RFC 7807) ---

// ProblemDetails represents RFC 7807 Problem Details
type ProblemDetails struct {
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Status   int                    `json:"status"`
	Detail   string                 `json:"detail,omitempty"`
	Instance string                 `json:"instance,omitempty"`
	Errors   map[string][]string    `json:"errors,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// ToProblemDetails converts AppError to RFC 7807 Problem Details
func (e *AppError) ToProblemDetails(baseURL string) ProblemDetails {
	return ProblemDetails{
		Type:   fmt.Sprintf("%s/errors/%s", baseURL, strings.ToLower(string(e.Code))),
		Title:  string(e.Code),
		Status: e.Status,
		Detail: e.Message,
		Errors: e.Errors,
	}
}

// ProblemDetailsHandler returns a handler that uses RFC 7807 format
func ProblemDetailsHandler(baseURL string, cfg ...Config) gin.HandlerFunc {
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}

	config.Transformer = func(c *gin.Context, err *AppError) interface{} {
		pd := err.ToProblemDetails(baseURL)
		pd.Instance = c.Request.URL.Path
		return pd
	}

	return Handler(config)
}

// --- Response Helpers ---

// JSON sends a successful JSON response
func JSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Paginated sends a paginated response
func Paginated(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta":    meta,
	})
}

// --- Debug Page Renderer ---

// DebugPageData holds data for debug error page
type DebugPageData struct {
	Title       string
	Message     string
	Code        string
	Status      int
	File        string
	Line        int
	Stack       []StackFrame
	RouteName   string
	RequestID   string
	TraceID     string
	Request     RequestInfo
	SQLQueries  []DebugSQLQuery
	RecentLogs  []DebugLogEntry
	Environment map[string]string
}

// RequestInfo holds request information for debug
type RequestInfo struct {
	Method  string
	URL     string
	Headers map[string]string
	Query   map[string]string
	Body    string
}

// DebugSQLQuery captures a SQL statement shown on the debug page.
type DebugSQLQuery struct {
	Time         string
	Duration     string
	RowsAffected int64
	Statement    string
	Error        string
}

// DebugLogEntry captures a recent log line shown on the debug page.
type DebugLogEntry struct {
	Time      string
	Level     string
	Channel   string
	Message   string
	RequestID string
	TraceID   string
	Context   string
}

// DebugStackFrames parses a stack trace into frames for external debug builders.
func DebugStackFrames(stack string) []StackFrame {
	return parseStackTrace(stack)
}

// RenderDebugPage renders an HTML debug page
func RenderDebugPage(c *gin.Context, err *AppError) {
	data := DebugPageData{
		Title:   string(err.Code),
		Message: err.Message,
		Code:    string(err.Code),
		Status:  err.Status,
		File:    err.File,
		Line:    err.Line,
		Stack:   parseStackTrace(err.Stack),
		Request: RequestInfo{
			Method:  c.Request.Method,
			URL:     c.Request.URL.String(),
			Headers: make(map[string]string),
			Query:   make(map[string]string),
		},
		Environment: map[string]string{
			"Go Version": runtime.Version(),
			"OS":         runtime.GOOS,
			"Arch":       runtime.GOARCH,
		},
	}

	// Collect headers
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			data.Request.Headers[k] = v[0]
		}
	}

	// Collect query params
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			data.Request.Query[k] = v[0]
		}
	}

	RenderDebugPageData(c, err.Status, data)
}

// RenderDebugPageData renders a debug page from pre-built data.
func RenderDebugPageData(c *gin.Context, status int, data DebugPageData) {
	html := renderDebugHTML(data)
	c.Data(status, "text/html; charset=utf-8", []byte(html))
}

// renderDebugHTML generates a beautiful modern debug HTML page
func renderDebugHTML(data DebugPageData) string {
	stackHTML := `<tr><td colspan="4" class="empty">No stack frames captured</td></tr>`
	if len(data.Stack) > 0 {
		stackHTML = ""
		for index, frame := range data.Stack {
			stackHTML += fmt.Sprintf(
				`<tr><td class="mono">#%d</td><td class="mono">%s</td><td class="mono">%s</td><td class="mono">:%d</td></tr>`,
				index, frame.Function, frame.File, frame.Line,
			)
		}
	}

	requestHTML := fmt.Sprintf(
		`<tr><td class="key">Method</td><td class="val mono">%s</td></tr>
<tr><td class="key">URL</td><td class="val mono">%s</td></tr>
<tr><td class="key">Route Name</td><td class="val mono">%s</td></tr>
<tr><td class="key">Request ID</td><td class="val mono">%s</td></tr>
<tr><td class="key">Trace ID</td><td class="val mono">%s</td></tr>`,
		data.Request.Method,
		data.Request.URL,
		emptyDash(data.RouteName),
		emptyDash(data.RequestID),
		emptyDash(data.TraceID),
	)

	headersHTML := ""
	for key, value := range data.Request.Headers {
		headersHTML += fmt.Sprintf(`<tr><td class="key mono">%s</td><td class="val mono">%s</td></tr>`, key, value)
	}
	if headersHTML == "" {
		headersHTML = `<tr><td colspan="2" class="empty">No request headers captured</td></tr>`
	}

	queryHTML := ""
	for key, value := range data.Request.Query {
		queryHTML += fmt.Sprintf(`<tr><td class="key mono">%s</td><td class="val mono">%s</td></tr>`, key, value)
	}
	if queryHTML == "" {
		queryHTML = `<tr><td colspan="2" class="empty">No query parameters</td></tr>`
	}

	sqlHTML := ""
	for _, query := range data.SQLQueries {
		sqlHTML += fmt.Sprintf(
			`<tr><td class="mono">%s</td><td class="mono">%s</td><td class="mono">%d</td><td class="mono sql">%s</td><td class="mono">%s</td></tr>`,
			emptyDash(query.Time),
			emptyDash(query.Duration),
			query.RowsAffected,
			emptyDash(query.Statement),
			emptyDash(query.Error),
		)
	}
	if sqlHTML == "" {
		sqlHTML = `<tr><td colspan="5" class="empty">No SQL statements captured for this request</td></tr>`
	}

	logsHTML := ""
	for _, entry := range data.RecentLogs {
		logsHTML += fmt.Sprintf(
			`<tr><td class="mono">%s</td><td class="mono">%s</td><td class="mono">%s</td><td>%s</td><td class="mono">%s</td><td class="mono">%s</td><td class="mono context">%s</td></tr>`,
			emptyDash(entry.Time),
			emptyDash(entry.Level),
			emptyDash(entry.Channel),
			emptyDash(entry.Message),
			emptyDash(entry.RequestID),
			emptyDash(entry.TraceID),
			emptyDash(entry.Context),
		)
	}
	if logsHTML == "" {
		logsHTML = `<tr><td colspan="7" class="empty">No recent logs correlated to this request</td></tr>`
	}

	envHTML := ""
	for key, value := range data.Environment {
		envHTML += fmt.Sprintf(`<tr><td class="key">%s</td><td class="val mono">%s</td></tr>`, key, value)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%d %s | ZGO</title>
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
	<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500&family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
	<style>
		:root {
			--bg-primary: #09111f;
			--bg-secondary: #111c2d;
			--bg-tertiary: #18253a;
			--text-primary: #e4e4ef;
			--text-secondary: #b6c0d3;
			--text-muted: #7b8aa8;
			--accent-red: #f87171;
			--accent-pink: #fb7185;
			--accent-purple: #c084fc;
			--accent-blue: #60a5fa;
			--accent-cyan: #38bdf8;
			--accent-green: #34d399;
			--border: #29405e;
			--radius: 12px;
		}
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: 'Inter', -apple-system, sans-serif; background: var(--bg-primary); color: var(--text-primary); line-height: 1.6; }
		code, .mono { font-family: 'JetBrains Mono', monospace; }
		.header { background: linear-gradient(135deg, #ef4444 0%%, #7f1d1d 100%%); padding: 48px 40px; }
		.header-inner { max-width: 1400px; margin: 0 auto; position: relative; }
		.error-code { font-size: 72px; font-weight: 700; margin-bottom: 8px; }
		.error-type { font-size: 14px; font-weight: 600; background: rgba(0,0,0,0.24); display: inline-block; padding: 6px 14px; border-radius: 999px; margin-bottom: 16px; font-family: 'JetBrains Mono', monospace; }
		.error-msg { font-size: 28px; font-weight: 500; max-width: 800px; }
		.main { max-width: 1400px; margin: 0 auto; padding: 32px 40px 56px; }
		.summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
		.summary-item { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); padding: 16px 18px; }
		.summary-item .label { color: var(--text-muted); font-size: 12px; text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 8px; }
		.summary-item .value { color: var(--text-primary); font-size: 15px; font-weight: 600; word-break: break-all; }
		.grid { display: grid; gap: 24px; grid-template-columns: repeat(12, 1fr); }
		.span-12 { grid-column: span 12; }
		.span-6 { grid-column: span 6; }
		.card { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
		.card-header { background: var(--bg-tertiary); padding: 14px 18px; font-size: 12px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-secondary); }
		table { width: 100%%; border-collapse: collapse; }
		tr { border-bottom: 1px solid var(--border); }
		tr:last-child { border-bottom: none; }
		td { padding: 14px 18px; font-size: 14px; vertical-align: top; }
		td.key { color: var(--accent-cyan); width: 220px; font-weight: 500; }
		td.val { color: var(--text-secondary); word-break: break-word; }
		td.empty { color: var(--text-muted); font-style: italic; }
		.sql, .context { white-space: pre-wrap; word-break: break-word; }
		.footer { text-align: center; padding: 40px; color: var(--text-muted); font-size: 13px; }
		.footer a { color: var(--accent-purple); text-decoration: none; }
		.footer .logo { font-size: 24px; font-weight: 700; margin-bottom: 8px; }
		@media (max-width: 960px) {
			.main { padding: 24px 18px 40px; }
			.header { padding: 36px 18px; }
			.span-6, .span-12 { grid-column: span 12; }
			td.key { width: 160px; }
		}
	</style>
</head>
<body>
	<div class="header">
		<div class="header-inner">
			<div class="error-code">%d</div>
			<div class="error-type">%s</div>
			<div class="error-msg">%s</div>
		</div>
	</div>

	<div class="main">
		<div class="summary">
			<div class="summary-item"><div class="label">Source</div><div class="value mono">%s:%d</div></div>
			<div class="summary-item"><div class="label">Route</div><div class="value mono">%s</div></div>
			<div class="summary-item"><div class="label">Request ID</div><div class="value mono">%s</div></div>
			<div class="summary-item"><div class="label">Trace ID</div><div class="value mono">%s</div></div>
		</div>

		<div class="grid">
			<div class="card span-12">
				<div class="card-header">Stack Trace</div>
				<table><thead><tr><td class="key mono">#</td><td class="key mono">Function</td><td class="key mono">File</td><td class="key mono">Line</td></tr></thead><tbody>%s</tbody></table>
			</div>

			<div class="card span-6">
				<div class="card-header">Request</div>
				<table>%s</table>
			</div>

			<div class="card span-6">
				<div class="card-header">Query Parameters</div>
				<table>%s</table>
			</div>

			<div class="card span-12">
				<div class="card-header">Request Headers</div>
				<table>%s</table>
			</div>

			<div class="card span-12">
				<div class="card-header">SQL Timeline</div>
				<table><thead><tr><td class="key mono">Time</td><td class="key mono">Duration</td><td class="key mono">Rows</td><td class="key mono">Statement</td><td class="key mono">Error</td></tr></thead><tbody>%s</tbody></table>
			</div>

			<div class="card span-12">
				<div class="card-header">Recent Logs</div>
				<table><thead><tr><td class="key mono">Time</td><td class="key mono">Level</td><td class="key mono">Channel</td><td class="key">Message</td><td class="key mono">Request ID</td><td class="key mono">Trace ID</td><td class="key mono">Context</td></tr></thead><tbody>%s</tbody></table>
			</div>

			<div class="card span-12">
				<div class="card-header">Environment</div>
				<table>%s</table>
			</div>
		</div>
	</div>

	<div class="footer">
		<div class="logo">ZGO</div>
		<div>Framework Debug Mode • <a href="https://github.com/zgiai/zgo" target="_blank">Documentation</a></div>
	</div>
</body>
</html>`,
		data.Status, data.Title,
		data.Status, data.Code, data.Message,
		emptyDash(data.File), data.Line,
		emptyDash(data.RouteName),
		emptyDash(data.RequestID),
		emptyDash(data.TraceID),
		stackHTML,
		requestHTML,
		queryHTML,
		headersHTML,
		sqlHTML,
		logsHTML,
		envHTML,
	)
}

// DebugHandler returns a handler that renders HTML debug pages
func DebugHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := toAppError(c.Errors.Last().Err)

			// Check if client wants JSON
			accept := c.GetHeader("Accept")
			if strings.Contains(accept, "application/json") {
				return // Let the normal handler deal with it
			}

			RenderDebugPage(c, err)
			c.Abort()
		}
	}
}

// PrettyJSON returns indented JSON for debugging
func PrettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func emptyDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
