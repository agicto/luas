package cache

import (
	"context"
	"sync"
	"time"
)

const (
	// DefaultMemoryMaxEntries bounds process-local cache cardinality.
	DefaultMemoryMaxEntries = 10_000
	// DefaultMemoryMaxBytes bounds retained key and value payload bytes. Go map
	// and entry metadata overhead are additional and intentionally not estimated.
	DefaultMemoryMaxBytes int64 = 64 << 20
)

// MemoryConfig controls the bounded process-local cache adapter. Zero values
// select conservative defaults; negative values are invalid.
type MemoryConfig struct {
	MaxEntries   int
	MaxBytes     int64
	MaxItemBytes int
}

// MemoryStats exposes bounded payload usage without revealing cache keys.
type MemoryStats struct {
	Entries      int
	PayloadBytes int64
	MaxEntries   int
	MaxBytes     int64
}

type memoryEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
	newer     *memoryEntry
	older     *memoryEntry
}

func (e *memoryEntry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && !now.Before(e.expiresAt)
}

// MemoryStore is a bounded, process-local LRU cache. It owns no background
// goroutines and copies values at the ownership boundary.
type MemoryStore struct {
	mu sync.Mutex

	entries map[string]*memoryEntry
	newest  *memoryEntry
	oldest  *memoryEntry

	maxEntries   int
	maxBytes     int64
	maxItemBytes int
	payloadBytes int64
	now          func() time.Time
}

// NewMemoryStore creates a bounded process-local cache.
func NewMemoryStore(config MemoryConfig) (*MemoryStore, error) {
	return newMemoryStore(config, time.Now)
}

func newMemoryStore(config MemoryConfig, now func() time.Time) (*MemoryStore, error) {
	if config.MaxEntries == 0 {
		config.MaxEntries = DefaultMemoryMaxEntries
	}
	if config.MaxBytes == 0 {
		config.MaxBytes = DefaultMemoryMaxBytes
	}
	if config.MaxItemBytes == 0 {
		config.MaxItemBytes = DefaultMaxItemBytes
		if int64(config.MaxItemBytes) > config.MaxBytes {
			config.MaxItemBytes = int(config.MaxBytes)
		}
	}
	if config.MaxEntries < 0 || config.MaxBytes < 0 || config.MaxItemBytes < 0 ||
		config.MaxEntries == 0 || config.MaxBytes == 0 || config.MaxItemBytes == 0 {
		return nil, ErrInvalidCapacity
	}
	if int64(config.MaxItemBytes) > config.MaxBytes || now == nil {
		return nil, ErrInvalidCapacity
	}

	return &MemoryStore{
		entries:      make(map[string]*memoryEntry),
		maxEntries:   config.MaxEntries,
		maxBytes:     config.MaxBytes,
		maxItemBytes: config.MaxItemBytes,
		now:          now,
	}, nil
}

// Get returns an owned copy and marks the entry as recently used.
func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}

	entry, ok := s.entries[key]
	if !ok {
		s.mu.Unlock()
		return nil, ErrCacheMiss
	}
	if entry.expired(s.now()) {
		s.removeLocked(entry)
		s.mu.Unlock()
		return nil, ErrCacheMiss
	}
	s.moveToNewestLocked(entry)
	value := entry.value
	s.mu.Unlock()
	return cloneBytes(value), nil
}

// Set stores a value with a positive TTL.
func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateTTL(ttl); err != nil {
		return err
	}
	return s.set(ctx, key, value, ttl)
}

// SetForever stores a value without expiry. Capacity eviction still applies.
func (s *MemoryStore) SetForever(ctx context.Context, key string, value []byte) error {
	return s.set(ctx, key, value, 0)
}

func (s *MemoryStore) set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateItemSize(value, s.maxItemBytes); err != nil {
		return err
	}
	weight := int64(len(key) + len(value))
	if weight > s.maxBytes {
		return ErrValueTooLarge
	}
	owned := cloneBytes(value)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if current, ok := s.entries[key]; ok {
		s.removeLocked(current)
	}
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = s.now().Add(ttl)
	}
	s.makeRoomLocked(weight)
	s.insertNewestLocked(&memoryEntry{key: key, value: owned, expiresAt: expiresAt})
	return nil
}

// Add atomically stores a value only when no live value exists.
func (s *MemoryStore) Add(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) (bool, error) {
	if err := validateContext(ctx); err != nil {
		return false, err
	}
	if err := validateKey(key); err != nil {
		return false, err
	}
	if err := validateTTL(ttl); err != nil {
		return false, err
	}
	if err := validateItemSize(value, s.maxItemBytes); err != nil {
		return false, err
	}
	weight := int64(len(key) + len(value))
	if weight > s.maxBytes {
		return false, ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	now := s.now()
	if current, ok := s.entries[key]; ok {
		if !current.expired(now) {
			s.moveToNewestLocked(current)
			return false, nil
		}
		s.removeLocked(current)
	}
	owned := cloneBytes(value)
	s.makeRoomLocked(weight)
	s.insertNewestLocked(&memoryEntry{
		key:       key,
		value:     owned,
		expiresAt: now.Add(ttl),
	})
	return true, nil
}

// Take atomically returns and removes a live value.
func (s *MemoryStore) Take(ctx context.Context, key string) ([]byte, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	entry, ok := s.entries[key]
	if !ok {
		s.mu.Unlock()
		return nil, ErrCacheMiss
	}
	s.removeLocked(entry)
	expired := entry.expired(s.now())
	value := entry.value
	s.mu.Unlock()
	if expired {
		return nil, ErrCacheMiss
	}
	return cloneBytes(value), nil
}

// Delete removes a key. Missing keys are not errors.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := validateKey(key); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if entry, ok := s.entries[key]; ok {
		s.removeLocked(entry)
	}
	return nil
}

// Clear removes all process-local entries. The shared Store seam deliberately
// omits global flushing because it is unsafe against a shared Redis keyspace.
func (s *MemoryStore) Clear(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.entries = make(map[string]*memoryEntry)
	s.newest = nil
	s.oldest = nil
	s.payloadBytes = 0
	return nil
}

// Stats returns active payload usage after opportunistically removing expiry.
func (s *MemoryStore) Stats() MemoryStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredLocked(s.now())
	return MemoryStats{
		Entries:      len(s.entries),
		PayloadBytes: s.payloadBytes,
		MaxEntries:   s.maxEntries,
		MaxBytes:     s.maxBytes,
	}
}

// Len returns the number of live entries.
func (s *MemoryStore) Len() int {
	return s.Stats().Entries
}

func (s *MemoryStore) makeRoomLocked(weight int64) {
	for len(s.entries) >= s.maxEntries || s.payloadBytes+weight > s.maxBytes {
		s.removeLocked(s.oldest)
	}
}

func (s *MemoryStore) removeExpiredLocked(now time.Time) {
	for _, entry := range s.entries {
		if entry.expired(now) {
			s.removeLocked(entry)
		}
	}
}

func (s *MemoryStore) insertNewestLocked(entry *memoryEntry) {
	entry.newer = nil
	entry.older = s.newest
	if s.newest != nil {
		s.newest.newer = entry
	} else {
		s.oldest = entry
	}
	s.newest = entry
	s.entries[entry.key] = entry
	s.payloadBytes += int64(len(entry.key) + len(entry.value))
}

func (s *MemoryStore) moveToNewestLocked(entry *memoryEntry) {
	if entry == s.newest {
		return
	}
	if entry.newer != nil {
		entry.newer.older = entry.older
	}
	if entry.older != nil {
		entry.older.newer = entry.newer
	} else {
		s.oldest = entry.newer
	}
	entry.newer = nil
	entry.older = s.newest
	s.newest.newer = entry
	s.newest = entry
}

func (s *MemoryStore) removeLocked(entry *memoryEntry) {
	if entry == nil {
		return
	}
	if entry.newer != nil {
		entry.newer.older = entry.older
	} else {
		s.newest = entry.older
	}
	if entry.older != nil {
		entry.older.newer = entry.newer
	} else {
		s.oldest = entry.newer
	}
	delete(s.entries, entry.key)
	s.payloadBytes -= int64(len(entry.key) + len(entry.value))
	entry.newer = nil
	entry.older = nil
}
