package audit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zgiai/luas/api/internal/domain"
)

type fakeBatchStore struct {
	mu        sync.Mutex
	single    []string
	batches   [][]string
	failBatch bool
	block     chan struct{}
}

func (s *fakeBatchStore) Create(_ context.Context, entry *domain.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.single = append(s.single, entry.Path)
	return nil
}

func (s *fakeBatchStore) CreateBatch(_ context.Context, entries []*domain.AuditLog) error {
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failBatch {
		return errors.New("batch rejected")
	}
	paths := make([]string, len(entries))
	for index, entry := range entries {
		paths[index] = entry.Path
	}
	s.batches = append(s.batches, paths)
	return nil
}

func (s *fakeBatchStore) written() (single []string, batched []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, batch := range s.batches {
		batched = append(batched, batch...)
	}
	return append([]string(nil), s.single...), batched
}

func TestAsyncWriterBatchesQueuedRecordsAndDrainsOnShutdown(t *testing.T) {
	store := &fakeBatchStore{}
	writer := newAsyncWriter(store)
	for _, path := range []string{"/a", "/b", "/c"} {
		require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: path}))
	}
	require.NoError(t, writer.Shutdown(context.Background()))

	single, batched := store.written()
	assert.Empty(t, single)
	assert.Equal(t, []string{"/a", "/b", "/c"}, batched)
}

func TestAsyncWriterWritesSynchronouslyAfterShutdown(t *testing.T) {
	store := &fakeBatchStore{}
	writer := newAsyncWriter(store)
	require.NoError(t, writer.Shutdown(context.Background()))

	require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: "/late"}))
	single, _ := store.written()
	assert.Equal(t, []string{"/late"}, single)
}

func TestAsyncWriterFallsBackToSynchronousWritesWhenTheQueueIsFull(t *testing.T) {
	store := &fakeBatchStore{block: make(chan struct{})}
	writer := newAsyncWriter(store)
	// The first record starts a flush that blocks, so the queue fills behind it.
	total := writerQueueSize + writerBatchSize + 10
	for index := 0; index < total; index++ {
		require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: "/r"}))
	}
	single, _ := store.written()
	assert.NotEmpty(t, single, "records beyond the queue are written synchronously, never dropped")
	close(store.block)
	require.NoError(t, writer.Shutdown(context.Background()))

	single, batched := store.written()
	assert.Equal(t, total, len(single)+len(batched))
}

func TestAsyncWriterRetriesRowsWhenABatchFails(t *testing.T) {
	store := &fakeBatchStore{failBatch: true}
	writer := newAsyncWriter(store)
	require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: "/x"}))
	require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: "/y"}))
	require.NoError(t, writer.Shutdown(context.Background()))

	single, batched := store.written()
	assert.Empty(t, batched)
	assert.ElementsMatch(t, []string{"/x", "/y"}, single)
}

func TestAsyncWriterShutdownHonorsItsDeadline(t *testing.T) {
	store := &fakeBatchStore{block: make(chan struct{})}
	writer := newAsyncWriter(store)
	require.NoError(t, writer.Write(context.Background(), &domain.AuditLog{Path: "/slow"}))
	time.Sleep(2 * writerFlushInterval) // let the flush start and block

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assert.Error(t, writer.Shutdown(ctx))
	close(store.block)
}
