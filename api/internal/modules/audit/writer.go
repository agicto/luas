package audit

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/zgiai/luas/api/internal/domain"
)

const (
	// writerQueueSize bounds records waiting in memory; a full queue writes synchronously instead.
	writerQueueSize = 4096
	// writerBatchSize and writerFlushInterval bound how long a record waits and how many rows one
	// INSERT carries.
	writerBatchSize     = 200
	writerFlushInterval = 50 * time.Millisecond
	writerWriteTimeout  = 10 * time.Second
)

// batchStore is the persistence the writer needs.
type batchStore interface {
	Create(ctx context.Context, entry *domain.AuditLog) error
	CreateBatch(ctx context.Context, entries []*domain.AuditLog) error
}

// asyncWriter takes request audit records off the response path and writes them in multi-row
// batches. It never drops a record it accepted: a full queue or a stopped writer falls back to a
// synchronous write, and Shutdown drains the queue. Records still queued when the process dies
// without a graceful shutdown are lost, which the audit contract accepts (best effort, not an
// immutable archive).
type asyncWriter struct {
	store   batchStore
	queue   chan *domain.AuditLog
	stop    chan struct{}
	done    chan struct{}
	mu      sync.RWMutex
	stopped bool
}

func newAsyncWriter(store batchStore) *asyncWriter {
	writer := &asyncWriter{
		store: store,
		queue: make(chan *domain.AuditLog, writerQueueSize),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go writer.run()
	return writer
}

// Write queues entry, or writes it synchronously when the queue is full or the writer stopped.
func (w *asyncWriter) Write(ctx context.Context, entry *domain.AuditLog) error {
	w.mu.RLock()
	if !w.stopped {
		select {
		case w.queue <- entry:
			w.mu.RUnlock()
			return nil
		default:
		}
	}
	w.mu.RUnlock()
	return w.store.Create(context.WithoutCancel(ctx), entry)
}

// Shutdown stops accepting records, writes everything queued, and waits until ctx ends.
func (w *asyncWriter) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.stop)
	}
	w.mu.Unlock()
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return errors.Join(ctx.Err(), errors.New("audit writer did not drain before shutdown"))
	}
}

func (w *asyncWriter) run() {
	defer close(w.done)
	ticker := time.NewTicker(writerFlushInterval)
	defer ticker.Stop()
	batch := make([]*domain.AuditLog, 0, writerBatchSize)
	flush := func() {
		if len(batch) > 0 {
			w.flush(batch)
			batch = make([]*domain.AuditLog, 0, writerBatchSize)
		}
	}
	for {
		select {
		case entry := <-w.queue:
			batch = append(batch, entry)
			if len(batch) == writerBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.stop:
			// Writers check stopped under the lock before sending, so the queue only shrinks now.
			for {
				select {
				case entry := <-w.queue:
					batch = append(batch, entry)
					if len(batch) == writerBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// flush writes one batch; if the batch insert fails it retries row by row, so one bad record
// cannot discard the others.
func (w *asyncWriter) flush(batch []*domain.AuditLog) {
	ctx, cancel := context.WithTimeout(context.Background(), writerWriteTimeout)
	defer cancel()
	err := w.store.CreateBatch(ctx, batch)
	if err == nil {
		return
	}
	slog.WarnContext(ctx, "audit.batch_write_failed", "records", len(batch), "err", err)
	for _, entry := range batch {
		if err := w.store.Create(ctx, entry); err != nil {
			slog.WarnContext(ctx, "audit.write_failed",
				"err", err, "method", entry.Method, "path", entry.Path, "request_id", entry.RequestID)
		}
	}
}
