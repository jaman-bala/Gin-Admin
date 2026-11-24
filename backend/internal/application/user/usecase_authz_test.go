package user

import (
	"context"
	"errors"
	"testing"

	domainUser "gin_auth_service/internal/domain/user"
	appErrors "gin_auth_service/pkg/errors"

	"uuid"
)

// --- fakes ---------------------------------------------------------------

// memRepo is an in-memory domainUser.Repository. GetID hands out copies, as a
// database does, so a mutation that is never persisted does not leak.
type memRepo struct {
	domainUser.Repository // unused methods panic on the nil interface
	users                 map[uuid.UUID]*domainUser.User
}

func newMemRepo(users ...*domainUser.User) *memRepo {
	r := &memRepo{users: map[uuid.UUID]*domainUser.User{}}
	for _, u := range users {
		c := *u
		r.users[u.ID] = &c
	}
	return r
}

func (r *memRepo) GetID(_ context.Context, id uuid.UUID) (*domainUser.User, error) {
	u, ok := r.users[id]
	if !ok || u.DeletedAt != nil {
		return nil, appErrors.ErrUserNotFound
	}
	c := *u
	return &c, nil
}

func (r *memRepo) Create(_ context.Context, u *domainUser.User) error {
	c := *u
	r.users[u.ID] = &c
	return nil
}

func (r *memRepo) Patch(_ context.Context, u *domainUser.User) error {
	c := *u
	r.users[u.ID] = &c
	return nil
}

func (r *memRepo) Delete(_ context.Context, id uuid.UUID) error {
	if u, ok := r.users[id]; ok {
		now := u.UpdatedAt
		u.DeletedAt = &now
	}
	return nil
}

func (r *memRepo) LockActiveSuperusers(context.Context) (int, error) {
	n := 0
	for _, u := range r.users {
		if u.Role == domainUser.RoleSuperuser && u.IsActive && u.DeletedAt == nil {
			n++
		}
	}
	return n, nil
}

// passTx runs fn directly: atomicity is the database's job (see the
// PostgreSQL integration test); here only the decisions are under test.
type passTx struct{}

func (passTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func newUC(users ...*domainUser.User) (UseCase, *memRepo) {
	repo := newMemRepo(users...)
	return NewUseCase(repo, passTx{}, nil, nil), repo
}

func person(role domainUser.Role, active bool) *domainUser.User {
	return &domainUser.User{ID: uuid.NewV7(), Phone: "+99650" + uuid.NewV7().String()[24:30], Password: "old-hash", Role: role, IsActive: active}
}

func actorOf(u *domainUser.User) Actor { return Actor{ID: u.ID, Role: u.Role} }

func str(s string) *string { return &s }
func flag(b bool) *bool    { return &b }

// --- Create --------------------------------------------------------------

func TestCreate_RoleCeiling(t *testing.T) {
	tests := []struct {
		name    string
		actor   domainUser.Role
		newRole string
		wantErr error
	}{
		{"admin creates user", domainUser.RoleAdmin, "user", nil},
		{"admin creates admin", domainUser.RoleAdmin, "admin", nil},
		{"admin cannot mint a superuser", domainUser.RoleAdmin, "superuser", appErrors.ErrForbiddenRole},
		{"superuser creates superuser", domainUser.RoleSuperuser, "superuser", nil},
		{"plain user cannot create anyone", domainUser.RoleUser, "user", appErrors.ErrForbiddenRole},
		{"unknown actor role cannot create", domainUser.Role(""), "user", appErrors.ErrForbiddenRole},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc, repo := newUC()
			_, err := uc.Create(context.Background(), Actor{ID: uuid.NewV7(), Role: tc.actor},
				UserRequestDTO{FirstName: "A", LastName: "B", Password: "Password123", Phone: "+996500111222", Role: tc.newRole}, nil)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && len(repo.users) != 0 {
				t.Errorf("a refused Create must not store anything, stored %d", len(repo.users))
			}
		})
	}
}

func TestCreate_PersistsTelegram(t *testing.T) {
	uc, repo := newUC()
	dto, err := uc.Create(context.Background(), Actor{Role: domainUser.RoleAdmin},
		UserRequestDTO{FirstName: "A", LastName: "B", Password: "Password123", Phone: "+996500111222", Role: "user", Telegram: "@alice"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := repo.users[dto.ID].Telegram; got != "@alice" {
		t.Errorf("stored telegram = %q; the field is accepted by the API and must not be silently dropped", got)
	}
}

// --- Patch ---------------------------------------------------------------

func TestPatch_AdminCannotTakeOverASuperuser(t *testing.T) {
	admin := person(domainUser.RoleAdmin, true)
	boss := person(domainUser.RoleSuperuser, true)
	other := person(domainUser.RoleSuperuser, true) // so the last-superuser rule is not what blocks it
	uc, repo := newUC(admin, boss, other)

	attacks := map[string]UserUpdateDTO{
		"reset password":     {Password: str("Hacked12345")},
		"block account":      {IsActive: flag(false)},
		"change role":        {Role: str("user")},
		"change phone/login": {Phone: str("+996999000111")},
		"harmless-looking":   {FirstName: str("x")},
	}
	for name, req := range attacks {
		t.Run(name, func(t *testing.T) {
			_, err := uc.Patch(context.Background(), actorOf(admin), boss.ID, req, nil)
			if !errors.Is(err, appErrors.ErrForbiddenRole) {
				t.Fatalf("err = %v, want ErrForbiddenRole", err)
			}
			got := repo.users[boss.ID]
			if got.Password != "old-hash" || !got.IsActive || got.Role != domainUser.RoleSuperuser || got.Phone != boss.Phone || got.FirstName != "" {
				t.Errorf("superuser was modified despite the refusal: %+v", got)
			}
		})
	}
}

func TestPatch_RoleCeiling(t *testing.T) {
	tests := []struct {
		name    string
		actor   domainUser.Role
		target  domainUser.Role
		newRole *string
		wantErr error
	}{
		{"admin edits user", domainUser.RoleAdmin, domainUser.RoleUser, nil, nil},
		{"admin edits peer admin", domainUser.RoleAdmin, domainUser.RoleAdmin, nil, nil},
		{"admin promotes user to admin", domainUser.RoleAdmin, domainUser.RoleUser, str("admin"), nil},
		{"admin cannot promote to superuser", domainUser.RoleAdmin, domainUser.RoleUser, str("superuser"), appErrors.ErrForbiddenRole},
		{"admin cannot promote self to superuser", domainUser.RoleAdmin, domainUser.RoleAdmin, str("superuser"), appErrors.ErrForbiddenRole},
		{"superuser edits admin", domainUser.RoleSuperuser, domainUser.RoleAdmin, nil, nil},
		{"superuser promotes admin to superuser", domainUser.RoleSuperuser, domainUser.RoleAdmin, str("superuser"), nil},
		{"plain user cannot edit anyone", domainUser.RoleUser, domainUser.RoleUser, nil, appErrors.ErrForbiddenRole},
		{"unknown actor role cannot edit", domainUser.Role("root"), domainUser.RoleUser, nil, appErrors.ErrForbiddenRole},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actor := person(tc.actor, true)
			target := person(tc.target, true)
			uc, repo := newUC(actor, target)

			_, err := uc.Patch(context.Background(), actorOf(actor), target.ID,
				UserUpdateDTO{FirstName: str("New"), Role: tc.newRole}, nil)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil && (repo.users[target.ID].FirstName != "" || repo.users[target.ID].Role != tc.target) {
				t.Errorf("refused Patch changed the target: %+v", repo.users[target.ID])
			}
		})
	}
}

func TestPatch_SelfPromotionIsBlocked(t *testing.T) {
	admin := person(domainUser.RoleAdmin, true)
	uc, repo := newUC(admin)

	_, err := uc.Patch(context.Background(), actorOf(admin), admin.ID, UserUpdateDTO{Role: str("superuser")}, nil)
	if !errors.Is(err, appErrors.ErrForbiddenRole) || repo.users[admin.ID].Role != domainUser.RoleAdmin {
		t.Fatalf("err = %v role = %v; an admin must not promote themselves", err, repo.users[admin.ID].Role)
	}
}

func TestPatch_LastSuperuserIsProtected(t *testing.T) {
	t.Run("only superuser cannot be demoted", func(t *testing.T) {
		su := person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(su)
		_, err := uc.Patch(context.Background(), actorOf(su), su.ID, UserUpdateDTO{Role: str("admin")}, nil)
		if !errors.Is(err, appErrors.ErrLastSuperuser) || repo.users[su.ID].Role != domainUser.RoleSuperuser {
			t.Fatalf("err = %v role = %v", err, repo.users[su.ID].Role)
		}
	})
	t.Run("only superuser cannot be deactivated", func(t *testing.T) {
		su := person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(su)
		_, err := uc.Patch(context.Background(), actorOf(su), su.ID, UserUpdateDTO{IsActive: flag(false)}, nil)
		if !errors.Is(err, appErrors.ErrLastSuperuser) || !repo.users[su.ID].IsActive {
			t.Fatalf("err = %v active = %v", err, repo.users[su.ID].IsActive)
		}
	})
	t.Run("an inactive superuser does not count", func(t *testing.T) {
		a, dormant := person(domainUser.RoleSuperuser, true), person(domainUser.RoleSuperuser, false)
		uc, _ := newUC(a, dormant)
		_, err := uc.Patch(context.Background(), actorOf(a), a.ID, UserUpdateDTO{Role: str("admin")}, nil)
		if !errors.Is(err, appErrors.ErrLastSuperuser) {
			t.Fatalf("err = %v, want ErrLastSuperuser", err)
		}
	})
	t.Run("harmless edit of the only superuser is fine", func(t *testing.T) {
		su := person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(su)
		if _, err := uc.Patch(context.Background(), actorOf(su), su.ID, UserUpdateDTO{FirstName: str("Root")}, nil); err != nil {
			t.Fatal(err)
		}
		if repo.users[su.ID].FirstName != "Root" {
			t.Error("edit was not applied")
		}
	})
	t.Run("demotion is allowed while another active superuser remains", func(t *testing.T) {
		a, b := person(domainUser.RoleSuperuser, true), person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(a, b)
		if _, err := uc.Patch(context.Background(), actorOf(a), b.ID, UserUpdateDTO{Role: str("admin")}, nil); err != nil {
			t.Fatal(err)
		}
		if repo.users[b.ID].Role != domainUser.RoleAdmin {
			t.Error("demotion was not applied")
		}
	})
}

func TestPatch_Errors(t *testing.T) {
	admin := person(domainUser.RoleAdmin, true)
	uc, _ := newUC(admin)

	if _, err := uc.Patch(context.Background(), actorOf(admin), uuid.NewV7(), UserUpdateDTO{FirstName: str("x")}, nil); !errors.Is(err, appErrors.ErrUserNotFound) {
		t.Errorf("unknown id: err = %v, want ErrUserNotFound", err)
	}
	if _, err := uc.Patch(context.Background(), actorOf(admin), uuid.Nil(), UserUpdateDTO{}, nil); !errors.Is(err, appErrors.ErrInvalidUUID) {
		t.Errorf("nil id: err = %v, want ErrInvalidUUID", err)
	}
}

// --- Delete --------------------------------------------------------------

func TestDelete_Rules(t *testing.T) {
	t.Run("admin cannot delete a superuser", func(t *testing.T) {
		admin, boss, other := person(domainUser.RoleAdmin, true), person(domainUser.RoleSuperuser, true), person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(admin, boss, other)
		if err := uc.Delete(context.Background(), actorOf(admin), boss.ID); !errors.Is(err, appErrors.ErrForbiddenRole) {
			t.Fatalf("err = %v, want ErrForbiddenRole", err)
		}
		if repo.users[boss.ID].DeletedAt != nil {
			t.Error("superuser was deleted despite the refusal")
		}
	})
	t.Run("admin deletes a user", func(t *testing.T) {
		admin, victim := person(domainUser.RoleAdmin, true), person(domainUser.RoleUser, true)
		uc, repo := newUC(admin, victim)
		if err := uc.Delete(context.Background(), actorOf(admin), victim.ID); err != nil {
			t.Fatal(err)
		}
		if repo.users[victim.ID].DeletedAt == nil {
			t.Error("user was not deleted")
		}
	})
	t.Run("last superuser cannot be deleted, even by themselves", func(t *testing.T) {
		su := person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(su)
		if err := uc.Delete(context.Background(), actorOf(su), su.ID); !errors.Is(err, appErrors.ErrLastSuperuser) {
			t.Fatalf("err = %v, want ErrLastSuperuser", err)
		}
		if repo.users[su.ID].DeletedAt != nil {
			t.Error("last superuser was deleted")
		}
	})
	t.Run("a superuser can be deleted while another remains", func(t *testing.T) {
		a, b := person(domainUser.RoleSuperuser, true), person(domainUser.RoleSuperuser, true)
		uc, repo := newUC(a, b)
		if err := uc.Delete(context.Background(), actorOf(a), b.ID); err != nil {
			t.Fatal(err)
		}
		if repo.users[b.ID].DeletedAt == nil {
			t.Error("superuser was not deleted")
		}
	})
	t.Run("plain user cannot delete anyone", func(t *testing.T) {
		u, victim := person(domainUser.RoleUser, true), person(domainUser.RoleUser, true)
		uc, _ := newUC(u, victim)
		if err := uc.Delete(context.Background(), actorOf(u), victim.ID); !errors.Is(err, appErrors.ErrForbiddenRole) {
			t.Fatalf("err = %v, want ErrForbiddenRole", err)
		}
	})
	t.Run("unknown and already deleted ids are not found, not silent successes", func(t *testing.T) {
		admin, victim := person(domainUser.RoleAdmin, true), person(domainUser.RoleUser, true)
		uc, _ := newUC(admin, victim)
		if err := uc.Delete(context.Background(), actorOf(admin), uuid.NewV7()); !errors.Is(err, appErrors.ErrUserNotFound) {
			t.Errorf("unknown id: err = %v, want ErrUserNotFound", err)
		}
		if err := uc.Delete(context.Background(), actorOf(admin), victim.ID); err != nil {
			t.Fatal(err)
		}
		if err := uc.Delete(context.Background(), actorOf(admin), victim.ID); !errors.Is(err, appErrors.ErrUserNotFound) {
			t.Errorf("second delete: err = %v, want ErrUserNotFound", err)
		}
	})
}
