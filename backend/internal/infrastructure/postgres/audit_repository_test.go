package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"gin_auth_service/internal/domain/auditlog"

	"github.com/jmoiron/sqlx"
	goose "github.com/pressly/goose/v3"
	"uuid"
)

// Integration tests against a real PostgreSQL. Run with e.g.:
//
//	POSTGRES_TEST_DSN='host=localhost port=5432 user=postgres password=... dbname=test sslmode=disable' \
//	go test ./internal/infrastructure/postgres/...
//
// They apply the repository's migrations (idempotent) and clean up their own rows.
func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping PostgreSQL integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db.DB, "../../../migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return db
}

// seedUser inserts a real user (audit_logs.user_id has a foreign key to it).
func seedUser(t *testing.T, db *sqlx.DB) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	phone := "+99650" + id.String()[24:30]
	if _, err := db.Exec(`INSERT INTO users (id, phone, password, role) VALUES ($1, $2, 'x', 'user')`, id, phone); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id) })
	return id
}

func auditEntry(marker string, user uuid.UUID) auditlog.AuditLog {
	return auditlog.AuditLog{
		UserID: user, Action: "POST", Entity: "Auth", EntityID: uuid.Nil(),
		ClientIP: "203.0.113.7", UserAgent: "test", Data: marker, Status: 200, CreatedAt: time.Now(),
	}
}

func cleanupAudit(t *testing.T, db *sqlx.DB, marker string) {
	t.Helper()
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM audit_logs WHERE data = $1`, marker) })
}

func countAudit(t *testing.T, db *sqlx.DB, marker string) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM audit_logs WHERE data = $1`, marker); err != nil {
		t.Fatal(err)
	}
	return n
}

// Unauthenticated requests (login, refresh, logout) have no user. They are
// represented by uuid.Nil in the domain and must be stored as NULL, otherwise
// the foreign key rejects them — and a rejected row sinks its whole batch.
func TestAuditCreateBatch_AnonymousEntryDoesNotSinkTheBatch(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)
	marker := "audit-it-" + uuid.NewV7().String()
	cleanupAudit(t, db, marker)
	user := seedUser(t, db)

	err := repo.CreateBatch(context.Background(), []auditlog.AuditLog{
		auditEntry(marker, user),
		auditEntry(marker, uuid.Nil()), // anonymous login attempt
		auditEntry(marker, user),
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if got := countAudit(t, db, marker); got != 3 {
		t.Fatalf("stored %d of 3 entries", got)
	}

	var nulls int
	if err := db.Get(&nulls, `SELECT COUNT(*) FROM audit_logs WHERE data = $1 AND user_id IS NULL`, marker); err != nil || nulls != 1 {
		t.Fatalf("NULL user_id rows = %d (err %v), want exactly the anonymous one", nulls, err)
	}
}

func TestAuditCreate_AnonymousEntry(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)
	marker := "audit-it-" + uuid.NewV7().String()
	cleanupAudit(t, db, marker)

	e := auditEntry(marker, uuid.Nil())
	if err := repo.Create(context.Background(), &e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := countAudit(t, db, marker); got != 1 {
		t.Fatalf("stored %d entries, want 1", got)
	}
}

// Rows with a NULL user_id already exist today (anonymous entries once
// written, and ON DELETE SET NULL). Reading them back must not fail.
func TestAuditRead_NullUserIDMapsToNil(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)
	marker := "audit-it-" + uuid.NewV7().String()
	cleanupAudit(t, db, marker)
	user := seedUser(t, db)

	if _, err := db.Exec(`INSERT INTO audit_logs (user_id, action, entity, entity_id, data, status) VALUES (NULL, 'POST', 'Auth', $1, $2, 200)`, uuid.NewV7(), marker); err != nil {
		t.Fatal(err)
	}
	entry := auditEntry(marker, user)
	if err := repo.Create(context.Background(), &entry); err != nil {
		t.Fatal(err)
	}

	// Page through everything; the listing must not error on the NULL row.
	logs, _, err := repo.GetAllPaginated(context.Background(), 1, 100, nil)
	if err != nil {
		t.Fatalf("GetAllPaginated with a NULL user_id row present: %v", err)
	}
	var sawNil, sawUser bool
	for _, l := range logs {
		if l.Data != marker {
			continue
		}
		switch l.UserID {
		case uuid.Nil():
			sawNil = true
		case user:
			sawUser = true
		}
	}
	if !sawNil || !sawUser {
		t.Errorf("anonymous row mapped to Nil = %v, user row preserved = %v", sawNil, sawUser)
	}
}
