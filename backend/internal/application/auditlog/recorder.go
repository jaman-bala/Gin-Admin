package auditlog

import (
	"context"
	"log/slog"
	"time"

	"gin_auth_service/internal/domain/auditlog"
)

// Recorder accepts audit entries without blocking the request path.
type Recorder interface {
	Record(entry *auditlog.AuditLog)
}

// AsyncRecorder buffers audit entries in memory and flushes them to the
// repository in batches from a background goroutine. This keeps audit writes
// off the request hot path: a slow database can never back-pressure request
// handling. If the buffer overflows, entries are dropped with a warning —
// audit here is operational logging, not a financial ledger.
type AsyncRecorder struct {
	repo          auditlog.Repository
	entries       chan *auditlog.AuditLog
	done          chan struct{}
	batchSize     int
	flushInterval time.Duration
}

// NewAsyncRecorder starts the background flusher.
// bufferSize bounds memory usage; batchSize/flushInterval control write cadence.
func NewAsyncRecorder(repo auditlog.Repository, bufferSize, batchSize int, flushInterval time.Duration) *AsyncRecorder {
	r := &AsyncRecorder{
		repo:          repo,
		entries:       make(chan *auditlog.AuditLog, bufferSize),
		done:          make(chan struct{}),
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
	go r.run()
	return r
}

// Record enqueues an entry without blocking. Must not be called after Close.
func (r *AsyncRecorder) Record(entry *auditlog.AuditLog) {
	select {
	case r.entries <- entry:
	default:
		slog.Warn("audit buffer full, dropping entry", "entity", entry.Entity, "action", entry.Action)
	}
}

func (r *AsyncRecorder) run() {
	defer close(r.done)

	ticker := time.NewTicker(r.flushInterval)
	defer ticker.Stop()

	pending := make([]auditlog.AuditLog, 0, r.batchSize)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := r.repo.CreateBatch(ctx, pending); err != nil {
			slog.Error("audit batch insert failed", "error", err, "count", len(pending))
		}
		cancel()
		pending = pending[:0]
	}

	for {
		select {
		case entry, ok := <-r.entries:
			if !ok {
				flush()
				return
			}
			pending = append(pending, *entry)
			if len(pending) >= r.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// Close flushes remaining entries and stops the background goroutine.
// Call after the HTTP server has stopped accepting requests.
func (r *AsyncRecorder) Close(ctx context.Context) error {
	close(r.entries)
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
