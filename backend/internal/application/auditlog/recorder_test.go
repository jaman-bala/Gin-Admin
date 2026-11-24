package auditlog

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gin_auth_service/internal/domain/auditlog"

	"uuid"
)

// poisonRepo rejects any batch containing an entry whose Entity is "poison",
// the way PostgreSQL rejects a whole multi-row INSERT for one bad row.
type poisonRepo struct {
	auditlog.Repository
	mu     sync.Mutex
	stored []string
}

var errPoison = errors.New("poisoned row")

func (r *poisonRepo) CreateBatch(_ context.Context, logs []auditlog.AuditLog) error {
	for _, l := range logs {
		if l.Entity == "poison" {
			return errPoison
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range logs {
		r.stored = append(r.stored, l.Entity)
	}
	return nil
}

func (r *poisonRepo) Create(_ context.Context, l *auditlog.AuditLog) error {
	if l.Entity == "poison" {
		return errPoison
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stored = append(r.stored, l.Entity)
	return nil
}

func TestAsyncRecorder_OneBadEntryDoesNotDropTheRestOfTheBatch(t *testing.T) {
	repo := &poisonRepo{}
	rec := NewAsyncRecorder(repo, 16, 10, time.Hour) // flush only on Close

	for _, e := range []string{"first", "poison", "last"} {
		rec.Record(&auditlog.AuditLog{Entity: e, UserID: uuid.Nil()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rec.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.stored) != 2 || repo.stored[0] != "first" || repo.stored[1] != "last" {
		t.Fatalf("stored = %v, want [first last]: only the bad entry may be lost", repo.stored)
	}
}
