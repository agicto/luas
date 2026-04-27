package logger

import (
	"context"
	"strings"
	"sync"
)

const defaultMemoryLimit = 200

// MemoryHandler stores a rolling in-memory buffer of recent log entries.
// It is used by the local exception center to surface nearby request logs.
type MemoryHandler struct {
	mu      sync.RWMutex
	limit   int
	entries []Entry
	next    int
	full    bool
}

var defaultMemoryHandler = NewMemoryHandler(defaultMemoryLimit)

// NewMemoryHandler creates a new in-memory ring buffer handler.
func NewMemoryHandler(limit int) *MemoryHandler {
	if limit < 1 {
		limit = defaultMemoryLimit
	}
	return &MemoryHandler{
		limit:   limit,
		entries: make([]Entry, limit),
	}
}

// DefaultMemoryHandler returns the shared recent-log buffer used by the framework.
func DefaultMemoryHandler() *MemoryHandler {
	return defaultMemoryHandler
}

func (h *MemoryHandler) Handle(_ context.Context, entry *Entry) error {
	if h == nil || entry == nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries[h.next] = cloneEntry(entry)
	h.next = (h.next + 1) % h.limit
	if h.next == 0 {
		h.full = true
	}
	return nil
}

func (h *MemoryHandler) Close() error {
	return nil
}

// Recent returns up to limit recent entries, newest first.
func (h *MemoryHandler) Recent(limit int) []Entry {
	return h.recentFiltered(limit, nil)
}

// RecentByRequest returns recent entries correlated to the given request or trace identifiers.
func (h *MemoryHandler) RecentByRequest(requestID, traceID string, limit int) []Entry {
	requestID = strings.TrimSpace(requestID)
	traceID = strings.TrimSpace(traceID)
	return h.recentFiltered(limit, func(entry Entry) bool {
		if requestID != "" && entry.RequestID == requestID {
			return true
		}
		return traceID != "" && entry.TraceID == traceID
	})
}

func (h *MemoryHandler) recentFiltered(limit int, filter func(Entry) bool) []Entry {
	if h == nil {
		return nil
	}
	if limit < 1 {
		limit = 20
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	total := h.next
	if h.full {
		total = h.limit
	}
	if total == 0 {
		return nil
	}

	result := make([]Entry, 0, min(limit, total))
	for i := 0; i < total && len(result) < limit; i++ {
		idx := h.next - 1 - i
		if idx < 0 {
			idx += h.limit
		}
		entry := cloneEntryValue(h.entries[idx])
		if entry.Time.IsZero() {
			continue
		}
		if filter != nil && !filter(entry) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func cloneEntry(entry *Entry) Entry {
	if entry == nil {
		return Entry{}
	}
	return cloneEntryValue(*entry)
}

func cloneEntryValue(entry Entry) Entry {
	entry.Context = copyMap(entry.Context)
	return entry
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
