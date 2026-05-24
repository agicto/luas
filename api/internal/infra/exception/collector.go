package exception

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type collectorKey struct{}

// SQLQuery captures a single SQL statement executed during a request.
type SQLQuery struct {
	Time         time.Time
	Duration     time.Duration
	Statement    string
	RowsAffected int64
	Error        string
}

// Collector stores request-scoped debug context for the local exception center.
type Collector struct {
	mu      sync.Mutex
	started time.Time
	method  string
	url     string
	headers map[string]string
	query   map[string]string
	sql     []SQLQuery
}

// NewCollector snapshots a request's basic shape before handler execution.
func NewCollector(req *http.Request) *Collector {
	collector := &Collector{
		started: time.Now(),
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
	if req == nil {
		return collector
	}

	collector.method = req.Method
	collector.url = req.URL.String()

	for key, values := range req.Header {
		if len(values) == 0 {
			continue
		}
		collector.headers[key] = values[0]
	}
	for key, values := range req.URL.Query() {
		if len(values) == 0 {
			continue
		}
		collector.query[key] = values[0]
	}

	return collector
}

// WithCollector stores the collector inside a request context.
func WithCollector(ctx context.Context, collector *Collector) context.Context {
	if collector == nil {
		return ctx
	}
	return context.WithValue(ctx, collectorKey{}, collector)
}

// FromContext returns the request-scoped collector if present.
func FromContext(ctx context.Context) *Collector {
	if ctx == nil {
		return nil
	}
	collector, _ := ctx.Value(collectorKey{}).(*Collector)
	return collector
}

// AddSQL records a SQL statement for later debug-page rendering.
func (c *Collector) AddSQL(started time.Time, duration time.Duration, statement string, rowsAffected int64, err error) {
	if c == nil {
		return
	}

	query := SQLQuery{
		Time:         started,
		Duration:     duration,
		Statement:    strings.TrimSpace(statement),
		RowsAffected: rowsAffected,
	}
	if err != nil {
		query.Error = err.Error()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.sql = append(c.sql, query)
}

// StartedAt returns the request start timestamp.
func (c *Collector) StartedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.started
}

// Method returns the original request method.
func (c *Collector) Method() string {
	if c == nil {
		return ""
	}
	return c.method
}

// URL returns the original request URL.
func (c *Collector) URL() string {
	if c == nil {
		return ""
	}
	return c.url
}

// Headers returns a copy of captured request headers.
func (c *Collector) Headers() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyStringMap(c.headers)
}

// Query returns a copy of captured query parameters.
func (c *Collector) Query() map[string]string {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyStringMap(c.query)
}

// SQL returns a copy of collected SQL timeline entries.
func (c *Collector) SQL() []SQLQuery {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	items := make([]SQLQuery, len(c.sql))
	copy(items, c.sql)
	return items
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	dup := make(map[string]string, len(values))
	for key, value := range values {
		dup[key] = value
	}
	return dup
}
