package user

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"testing"

	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/infrastructure/postgres"
	appErrors "gin_auth_service/pkg/errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	goose "github.com/pressly/goose/v3"
	"uuid"
)

// Integration test against a real PostgreSQL (POSTGRES_TEST_DSN). It creates
// and drops its own throw-away database: the "last superuser" rule counts
// every superuser in the database, so sharing one would make it flaky.
func newIsolatedDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping PostgreSQL integration test")
	}
	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	name := "usertest_" + uuid.NewV7().String()[24:]
	if _, err := admin.Exec(fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
		_ = admin.Close()
		t.Skipf("cannot create a throw-away database (needs CREATEDB): %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name))
		_ = admin.Close()
	})

	dbDSN := regexp.MustCompile(`dbname=\S+`).ReplaceAllString(dsn, "dbname="+name)
	db, err := sqlx.Connect("postgres", dbDSN)
	if err != nil {
		t.Fatalf("connect to %s: %v", name, err)
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

func seedSuperuser(t *testing.T, db *sqlx.DB) *domainUser.User {
	t.Helper()
	id := uuid.NewV7()
	phone := "+99650" + id.String()[24:30]
	if _, err := db.Exec(`INSERT INTO users (id, phone, password, role) VALUES ($1, $2, 'x', 'superuser')`, id, phone); err != nil {
		t.Fatal(err)
	}
	return &domainUser.User{ID: id, Role: domainUser.RoleSuperuser, IsActive: true}
}

func activeSuperusers(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM users WHERE role='superuser' AND is_active AND deleted_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	return n
}

// Two superusers act at the same moment, each removing the other. Without
// serialization both see "another superuser exists", both succeed, and the
// system is left with none. Exactly one must win.
func TestLastSuperuser_ConcurrentRemovalNeverLeavesZero(t *testing.T) {
	db := newIsolatedDB(t)
	uc := NewUseCase(postgres.NewUserRepository(db), postgres.NewTxManager(db), nil, nil)
	a, b := seedSuperuser(t, db), seedSuperuser(t, db)

	actions := map[string]func(actor Actor, target uuid.UUID) error{
		"demote": func(actor Actor, target uuid.UUID) error {
			role := "admin"
			_, err := uc.Patch(context.Background(), actor, target, UserUpdateDTO{Role: &role}, nil)
			return err
		},
		"deactivate": func(actor Actor, target uuid.UUID) error {
			off := false
			_, err := uc.Patch(context.Background(), actor, target, UserUpdateDTO{IsActive: &off}, nil)
			return err
		},
		"delete": func(actor Actor, target uuid.UUID) error {
			return uc.Delete(context.Background(), actor, target)
		},
	}

	for name, act := range actions {
		t.Run(name, func(t *testing.T) {
			for round := 0; round < 25; round++ {
				if _, err := db.Exec(`UPDATE users SET role='superuser', is_active=true, deleted_at=NULL, version=version+1 WHERE id IN ($1, $2)`,
					a.ID, b.ID); err != nil {
					t.Fatal(err)
				}

				var wg sync.WaitGroup
				start := make(chan struct{})
				errs := make([]error, 2)
				for i, pair := range [][2]*domainUser.User{{a, b}, {b, a}} {
					wg.Add(1)
					go func() {
						defer wg.Done()
						<-start
						errs[i] = act(Actor{ID: pair[0].ID, Role: domainUser.RoleSuperuser}, pair[1].ID)
					}()
				}
				close(start)
				wg.Wait()

				if n := activeSuperusers(t, db); n != 1 {
					t.Fatalf("round %d: %d active superusers left (errs: %v, %v), want exactly 1", round, n, errs[0], errs[1])
				}
				succeeded := 0
				for _, err := range errs {
					switch {
					case err == nil:
						succeeded++
					case errors.Is(err, appErrors.ErrLastSuperuser), errors.Is(err, appErrors.ErrStaleWrite), errors.Is(err, appErrors.ErrUserNotFound):
						// the loser: refused by the guard, or its target was already gone/changed
					default:
						t.Fatalf("round %d: unexpected error %v", round, err)
					}
				}
				if succeeded != 1 {
					t.Fatalf("round %d: %d requests succeeded, want exactly 1 (errs: %v, %v)", round, succeeded, errs[0], errs[1])
				}
			}
		})
	}
}

// Mixed real-database check of the rules that do not need concurrency.
func TestRoleRules_AgainstRealDatabase(t *testing.T) {
	db := newIsolatedDB(t)
	uc := NewUseCase(postgres.NewUserRepository(db), postgres.NewTxManager(db), nil, nil)
	boss, other := seedSuperuser(t, db), seedSuperuser(t, db)
	admin := Actor{ID: uuid.NewV7(), Role: domainUser.RoleAdmin}
	ctx := context.Background()

	pw := "Hacked12345"
	if _, err := uc.Patch(ctx, admin, boss.ID, UserUpdateDTO{Password: &pw}, nil); !errors.Is(err, appErrors.ErrForbiddenRole) {
		t.Fatalf("admin resetting a superuser's password: err = %v, want ErrForbiddenRole", err)
	}
	var stored string
	if err := db.Get(&stored, `SELECT password FROM users WHERE id=$1`, boss.ID); err != nil || stored != "x" {
		t.Fatalf("superuser's password changed despite the refusal (%q, %v)", stored, err)
	}
	if err := uc.Delete(ctx, admin, boss.ID); !errors.Is(err, appErrors.ErrForbiddenRole) {
		t.Fatalf("admin deleting a superuser: err = %v, want ErrForbiddenRole", err)
	}
	if err := uc.Delete(ctx, Actor{ID: boss.ID, Role: domainUser.RoleSuperuser}, other.ID); err != nil {
		t.Fatalf("superuser deleting another superuser: %v", err)
	}
	if err := uc.Delete(ctx, Actor{ID: boss.ID, Role: domainUser.RoleSuperuser}, boss.ID); !errors.Is(err, appErrors.ErrLastSuperuser) {
		t.Fatalf("deleting the last superuser: err = %v, want ErrLastSuperuser", err)
	}
	if err := uc.Delete(ctx, Actor{ID: boss.ID, Role: domainUser.RoleSuperuser}, other.ID); !errors.Is(err, appErrors.ErrUserNotFound) {
		t.Fatalf("deleting an already deleted user: err = %v, want ErrUserNotFound", err)
	}
}
