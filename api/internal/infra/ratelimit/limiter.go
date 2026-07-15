package ratelimit

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zgiai/luas/api/pkg/response"
)

// DefaultMaxBuckets bounds attacker-controlled identity cardinality per store.
const DefaultMaxBuckets = 10_000

// Limiter atomically evaluates and consumes one rate-limit decision.
type Limiter interface {
	Take(ctx context.Context, key string) (allowed bool, remaining int, resetAt time.Time)
}

// Config holds rate limiter configuration
type Config struct {
	// Max number of requests allowed
	Max int

	// Duration for the rate limit window
	Duration time.Duration

	// MaxBuckets bounds process-local identity cardinality.
	MaxBuckets int

	// KeyFunc extracts the rate limit key from request
	KeyFunc func(*gin.Context) string

	// ErrorHandler handles rate limit exceeded
	ErrorHandler func(*gin.Context, time.Time)

	// SkipFunc returns true to skip rate limiting for a request
	SkipFunc func(*gin.Context) bool

	// Store is the underlying storage (default: memory)
	Store Limiter

	// SuppressHeaders omits quota diagnostics from successful responses.
	// Sensitive endpoints can use this to avoid advertising bucket state.
	SuppressHeaders bool
}

// DefaultConfig returns default rate limiter configuration
func DefaultConfig() Config {
	return Config{
		Max:        60,
		Duration:   time.Minute,
		MaxBuckets: DefaultMaxBuckets,
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
		ErrorHandler: func(c *gin.Context, resetAt time.Time) {
			retryAfter := int(math.Ceil(time.Until(resetAt).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			response.AbortWithCode(c, http.StatusTooManyRequests, response.ErrorCodeRateLimited, "Too many requests")
		},
	}
}

// MemoryStore implements a bounded, process-local fixed-window limiter.
type MemoryStore struct {
	mu          sync.Mutex
	entries     map[string]*entry
	mostRecent  *entry
	leastRecent *entry
	max         int
	window      time.Duration

	maxBuckets    int
	now           func() time.Time
	cleanupEvery  time.Duration
	nextCleanupAt time.Time
}

type entry struct {
	key      string
	hits     int
	resetAt  time.Time
	previous *entry
	next     *entry
}

// MemoryStoreOption configures process-local limiter resource policy.
type MemoryStoreOption func(*MemoryStore)

// WithMaxBuckets caps the number of active identity buckets retained by one
// store. Once full, the least recently used bucket is evicted.
func WithMaxBuckets(max int) MemoryStoreOption {
	return func(store *MemoryStore) {
		if max > 0 {
			store.maxBuckets = max
		}
	}
}

func withClock(now func() time.Time) MemoryStoreOption {
	return func(store *MemoryStore) {
		if now != nil {
			store.now = now
		}
	}
}

// NewMemoryStore creates a bounded in-memory rate limiter store. Expired
// buckets are removed opportunistically, so each rule owns no background
// goroutine and needs no shutdown hook.
func NewMemoryStore(max int, window time.Duration, options ...MemoryStoreOption) *MemoryStore {
	defaults := DefaultConfig()
	if max <= 0 {
		max = defaults.Max
	}
	if window <= 0 {
		window = defaults.Duration
	}

	cleanupEvery := window
	if cleanupEvery > time.Minute {
		cleanupEvery = time.Minute
	}

	s := &MemoryStore{
		entries:      make(map[string]*entry),
		max:          max,
		window:       window,
		maxBuckets:   DefaultMaxBuckets,
		now:          time.Now,
		cleanupEvery: cleanupEvery,
	}
	for _, option := range options {
		option(s)
	}
	s.nextCleanupAt = s.now().Add(s.cleanupEvery)

	return s
}

// Take atomically checks the quota and records the hit. A denied request still
// refreshes recency so a hot abusive identity is not preferentially evicted.
func (s *MemoryStore) Take(ctx context.Context, key string) (bool, int, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if !now.Before(s.nextCleanupAt) {
		s.removeExpired(now)
		s.nextCleanupAt = now.Add(s.cleanupEvery)
	}

	e, exists := s.entries[key]
	if exists && !now.Before(e.resetAt) {
		s.removeEntry(key, e)
		e = nil
		exists = false
	}

	if !exists {
		if len(s.entries) >= s.maxBuckets {
			s.evictLeastRecentlyUsed()
		}
		e = &entry{key: key, hits: 1, resetAt: now.Add(s.window)}
		s.addMostRecent(e)
		s.entries[key] = e
		return true, max(0, s.max-1), e.resetAt
	}

	s.moveToMostRecent(e)
	if e.hits >= s.max {
		return false, 0, e.resetAt
	}

	e.hits++
	return true, max(0, s.max-e.hits), e.resetAt
}

func (s *MemoryStore) removeExpired(now time.Time) {
	for key, current := range s.entries {
		if !now.Before(current.resetAt) {
			s.removeEntry(key, current)
		}
	}
}

func (s *MemoryStore) evictLeastRecentlyUsed() {
	if s.leastRecent == nil {
		return
	}
	s.removeEntry(s.leastRecent.key, s.leastRecent)
}

func (s *MemoryStore) removeEntry(key string, current *entry) {
	delete(s.entries, key)
	if current.previous != nil {
		current.previous.next = current.next
	} else {
		s.mostRecent = current.next
	}
	if current.next != nil {
		current.next.previous = current.previous
	} else {
		s.leastRecent = current.previous
	}
	current.previous = nil
	current.next = nil
}

func (s *MemoryStore) addMostRecent(current *entry) {
	current.next = s.mostRecent
	if s.mostRecent != nil {
		s.mostRecent.previous = current
	} else {
		s.leastRecent = current
	}
	s.mostRecent = current
}

func (s *MemoryStore) moveToMostRecent(current *entry) {
	if s.mostRecent == current {
		return
	}
	if current.previous != nil {
		current.previous.next = current.next
	}
	if current.next != nil {
		current.next.previous = current.previous
	} else {
		s.leastRecent = current.previous
	}
	current.previous = nil
	current.next = nil
	s.addMostRecent(current)
}

// --- Middleware Functions ---

// Middleware creates a rate limiting middleware with the given config
func Middleware(cfg Config) gin.HandlerFunc {
	defaults := DefaultConfig()
	if cfg.Max <= 0 {
		cfg.Max = defaults.Max
	}
	if cfg.Duration <= 0 {
		cfg.Duration = defaults.Duration
	}
	if cfg.MaxBuckets <= 0 {
		cfg.MaxBuckets = DefaultMaxBuckets
	}
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore(cfg.Max, cfg.Duration, WithMaxBuckets(cfg.MaxBuckets))
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaults.KeyFunc
	}
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = defaults.ErrorHandler
	}

	return func(c *gin.Context) {
		// Check if should skip
		if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
			c.Next()
			return
		}

		key := cfg.KeyFunc(c)

		// Check and consume in one store operation so concurrent requests
		// cannot pass independently before recording their hits.
		allowed, remaining, resetAt := cfg.Store.Take(c.Request.Context(), key)
		if !allowed {
			cfg.ErrorHandler(c, resetAt)
			c.Abort()
			return
		}

		if !cfg.SuppressHeaders {
			c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		}

		c.Next()
	}
}

// PerMinute creates a middleware that limits requests per minute
func PerMinute(max int) gin.HandlerFunc {
	cfg := DefaultConfig()
	cfg.Max = max
	cfg.Duration = time.Minute
	return Middleware(cfg)
}

// PerHour creates a middleware that limits requests per hour
func PerHour(max int) gin.HandlerFunc {
	cfg := DefaultConfig()
	cfg.Max = max
	cfg.Duration = time.Hour
	return Middleware(cfg)
}

// PerDay creates a middleware that limits requests per day
func PerDay(max int) gin.HandlerFunc {
	cfg := DefaultConfig()
	cfg.Max = max
	cfg.Duration = 24 * time.Hour
	return Middleware(cfg)
}

// PerSecond creates a middleware that limits requests per second
func PerSecond(max int) gin.HandlerFunc {
	cfg := DefaultConfig()
	cfg.Max = max
	cfg.Duration = time.Second
	return Middleware(cfg)
}

// WithKeyFunc sets a custom key function for rate limiting
func WithKeyFunc(keyFunc func(*gin.Context) string) func(*Config) {
	return func(cfg *Config) {
		cfg.KeyFunc = keyFunc
	}
}

// WithSkipFunc sets a skip function for rate limiting
func WithSkipFunc(skipFunc func(*gin.Context) bool) func(*Config) {
	return func(cfg *Config) {
		cfg.SkipFunc = skipFunc
	}
}

// Custom creates a custom rate limiter middleware
func Custom(max int, duration time.Duration, opts ...func(*Config)) gin.HandlerFunc {
	cfg := Config{
		Max:          max,
		Duration:     duration,
		ErrorHandler: DefaultConfig().ErrorHandler,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return Middleware(cfg)
}

// ByUser creates a rate limiter keyed by user ID
func ByUser(max int, duration time.Duration, userIDFunc func(*gin.Context) string) gin.HandlerFunc {
	return Custom(max, duration, WithKeyFunc(userIDFunc))
}

// ByRoute creates a rate limiter keyed by route + IP
func ByRoute(max int, duration time.Duration) gin.HandlerFunc {
	return Custom(max, duration, WithKeyFunc(func(c *gin.Context) string {
		return c.FullPath() + ":" + c.ClientIP()
	}))
}

// ByAPIKey creates a rate limiter keyed by API key
func ByAPIKey(max int, duration time.Duration, headerName string) gin.HandlerFunc {
	return Custom(max, duration, WithKeyFunc(func(c *gin.Context) string {
		apiKey := c.GetHeader(headerName)
		if apiKey == "" {
			return c.ClientIP()
		}
		return apiKey
	}))
}

// Throttle is an alias for PerMinute.
func Throttle(maxPerMinute int) gin.HandlerFunc {
	return PerMinute(maxPerMinute)
}

// ThrottleWithDecay creates a rate limiter with decay (sliding window)
func ThrottleWithDecay(max int, decayMinutes int) gin.HandlerFunc {
	return Custom(max, time.Duration(decayMinutes)*time.Minute)
}
