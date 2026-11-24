package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	tokenUC "gin_auth_service/internal/application/token"
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/internal/pkg/jwt"
	appErrors "gin_auth_service/pkg/errors"

	"golang.org/x/crypto/bcrypt"
	"uuid"
)

const testPassword = "Password123"

// --- fakes ---------------------------------------------------------------

type fakeUserRepo struct {
	byPhone map[string]*domainUser.User
	byID    map[uuid.UUID]*domainUser.User
	findErr error // returned by FindByPhone when set
	getErr  error // returned by GetID when set
}

func newFakeUserRepo(users ...*domainUser.User) *fakeUserRepo {
	r := &fakeUserRepo{byPhone: map[string]*domainUser.User{}, byID: map[uuid.UUID]*domainUser.User{}}
	for _, u := range users {
		r.byPhone[u.Phone] = u
		r.byID[u.ID] = u
	}
	return r
}

func (r *fakeUserRepo) FindByPhone(_ context.Context, phone string) (*domainUser.User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if u, ok := r.byPhone[phone]; ok {
		return u, nil
	}
	return nil, appErrors.ErrUserNotFound
}

func (r *fakeUserRepo) GetID(_ context.Context, id uuid.UUID) (*domainUser.User, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, appErrors.ErrUserNotFound
}

func (r *fakeUserRepo) Create(context.Context, *domainUser.User) error { panic("unused") }
func (r *fakeUserRepo) GetWithFilters(context.Context, domainUser.FilterParams) ([]*domainUser.User, int, error) {
	panic("unused")
}
func (r *fakeUserRepo) Patch(context.Context, *domainUser.User) error { panic("unused") }
func (r *fakeUserRepo) PatchFields(context.Context, uuid.UUID, domainUser.PatchData) error {
	panic("unused")
}
func (r *fakeUserRepo) Delete(context.Context, uuid.UUID) error { panic("unused") }
func (r *fakeUserRepo) LockActiveSuperusers(context.Context) (int, error) { panic("unused") }

// memTokenRepo is an in-memory domain/token.Repository (Redis stand-in).
type memTokenRepo struct {
	mu     sync.Mutex
	keys   map[string]bool
	setErr error
}

func (r *memTokenRepo) Set(_ context.Context, key string, _ any, _ time.Duration) error {
	if r.setErr != nil {
		return r.setErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key] = true
	return nil
}
func (r *memTokenRepo) Delete(context.Context, string) error { return nil }
func (r *memTokenRepo) Exists(_ context.Context, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.keys[key], nil
}

type fakePhotoResolver struct {
	url string
	err error
}

func (f fakePhotoResolver) GetFullURL(_ context.Context, name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.url + name, nil
}

// --- helpers -------------------------------------------------------------

type env struct {
	uc     UseCase
	users  *fakeUserRepo
	tokens *memTokenRepo
	svc    tokenUC.UseCase
}

func newEnv(t *testing.T, photo PhotoResolver, users ...*domainUser.User) *env {
	t.Helper()
	repo := newFakeUserRepo(users...)
	tokens := &memTokenRepo{keys: map[string]bool{}}
	svc := tokenUC.NewUseCase(tokens, jwt.NewJWTService("unit-test-secret-key-0123456789abcdef"))
	if photo == nil {
		photo = fakePhotoResolver{url: "https://cdn.test/"}
	}
	return &env{
		uc:     NewUseCase(repo, svc, photo, 15*time.Minute, time.Hour),
		users:  repo,
		tokens: tokens,
		svc:    svc,
	}
}

func newUser(t *testing.T, phone string, role domainUser.Role, active bool) *domainUser.User {
	t.Helper()
	// MinCost keeps the suite fast; CheckPassword reads the cost from the hash,
	// so the code under test behaves identically.
	hb, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	h := string(hb)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return &domainUser.User{ID: uuid.NewV7(), Phone: phone, Password: h, Role: role, IsActive: active}
}

func login(t *testing.T, e *env, phone string) *LoginResponseDTO {
	t.Helper()
	resp, err := e.uc.Login(context.Background(), LoginRequestDTO{Phone: phone, Password: testPassword})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return resp
}

// --- Login ---------------------------------------------------------------

func TestLogin(t *testing.T) {
	ctx := context.Background()
	active := newUser(t, "+996500500500", domainUser.RoleAdmin, true)
	blocked := newUser(t, "+996500500501", domainUser.RoleUser, false)
	e := newEnv(t, nil, active, blocked)

	tests := []struct {
		name    string
		req     LoginRequestDTO
		wantErr error
	}{
		{"ok", LoginRequestDTO{Phone: "+996500500500", Password: testPassword}, nil},
		{"phone is normalized (no plus, spaces, dashes)", LoginRequestDTO{Phone: "996 500-500 500", Password: testPassword}, nil},
		{"wrong password", LoginRequestDTO{Phone: "+996500500500", Password: "wrong"}, appErrors.ErrInvalidCredentials},
		{"empty password", LoginRequestDTO{Phone: "+996500500500", Password: ""}, appErrors.ErrInvalidCredentials},
		{"unknown phone looks like wrong password", LoginRequestDTO{Phone: "+996700000000", Password: testPassword}, appErrors.ErrInvalidCredentials},
		{"blocked account", LoginRequestDTO{Phone: "+996500500501", Password: testPassword}, appErrors.ErrAccountBlocked},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := e.uc.Login(ctx, tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if resp != nil {
					t.Errorf("no tokens may be issued on failure, got %+v", resp)
				}
				return
			}
			if resp.AccessToken == "" || resp.RefreshToken == "" {
				t.Fatalf("empty tokens in %+v", resp)
			}
			info, err := e.svc.GetTokenInfo(ctx, resp.AccessToken)
			if err != nil {
				t.Fatalf("issued access token invalid: %v", err)
			}
			if info.UserID != active.ID.String() || info.Role != "admin" || !info.IsActive || info.Type != "access" {
				t.Errorf("unexpected claims %+v", info)
			}
		})
	}
}

// TestLogin_BlockedStatusIsReportedBeforePasswordCheck pins the CURRENT
// behaviour: is_active is checked before the password, so anyone who knows a
// blocked user's phone learns the account is blocked without the password.
// If that disclosure is ever closed (check the password first), update this test.
func TestLogin_BlockedStatusIsReportedBeforePasswordCheck(t *testing.T) {
	blocked := newUser(t, "+996500500501", domainUser.RoleUser, false)
	e := newEnv(t, nil, blocked)

	resp, err := e.uc.Login(context.Background(), LoginRequestDTO{Phone: blocked.Phone, Password: "wrong"})
	if !errors.Is(err, appErrors.ErrAccountBlocked) || resp != nil {
		t.Fatalf("got %+v, %v; want nil, ErrAccountBlocked", resp, err)
	}
}

func TestLogin_RepositoryFailureIsNotInvalidCredentials(t *testing.T) {
	e := newEnv(t, nil)
	boom := errors.New("db down")
	e.users.findErr = boom

	_, err := e.uc.Login(context.Background(), LoginRequestDTO{Phone: "+996500500500", Password: testPassword})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap the db error", err)
	}
	if errors.Is(err, appErrors.ErrInvalidCredentials) {
		t.Error("a DB outage must surface as 5xx, not as 'wrong password'")
	}
}

// --- Refresh -------------------------------------------------------------

func TestRefreshToken_RotatesAndRejectsReplay(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleAdmin, true)
	u.Photo = "me.png"
	e := newEnv(t, nil, u)
	first := login(t, e, u.Phone)

	resp, err := e.uc.RefreshToken(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.RefreshToken == first.RefreshToken {
		t.Errorf("a fresh, different pair is expected: %+v", resp)
	}
	if resp.User.ID != u.ID || resp.User.Photo != "https://cdn.test/me.png" {
		t.Errorf("unexpected user payload: %+v", resp.User)
	}
	if !resp.ExpiresAt.After(time.Now()) {
		t.Errorf("ExpiresAt %v must be in the future", resp.ExpiresAt)
	}

	// Replaying the old refresh token must fail: that is what rotation is for.
	if _, err := e.uc.RefreshToken(ctx, first.RefreshToken); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("replayed refresh token: err = %v, want ErrInvalidToken", err)
	}
	// The new one keeps working.
	if _, err := e.uc.RefreshToken(ctx, resp.RefreshToken); err != nil {
		t.Errorf("rotated refresh token should be usable: %v", err)
	}
}

func TestRefreshToken_Rejects(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	pair := login(t, e, u.Phone)

	t.Run("access token is not a refresh token", func(t *testing.T) {
		if _, err := e.uc.RefreshToken(ctx, pair.AccessToken); !errors.Is(err, appErrors.ErrInvalidToken) {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})
	t.Run("garbage", func(t *testing.T) {
		if _, err := e.uc.RefreshToken(ctx, "garbage"); !errors.Is(err, appErrors.ErrInvalidToken) {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})
	t.Run("user blocked after login", func(t *testing.T) {
		u.IsActive = false
		defer func() { u.IsActive = true }()
		if _, err := e.uc.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, appErrors.ErrAccountBlocked) {
			t.Errorf("err = %v, want ErrAccountBlocked", err)
		}
	})
	t.Run("user deleted after login", func(t *testing.T) {
		delete(e.users.byID, u.ID)
		defer func() { e.users.byID[u.ID] = u }()
		if _, err := e.uc.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, appErrors.ErrUserNotFound) {
			t.Errorf("err = %v, want ErrUserNotFound", err)
		}
	})
}

func TestRefreshToken_AbortsWhenOldTokenCannotBeBlacklisted(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	pair := login(t, e, u.Phone)

	e.tokens.setErr = errors.New("redis down")
	resp, err := e.uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err == nil || resp != nil {
		t.Fatalf("got %+v, %v; issuing a new pair while the old one stays valid defeats rotation", resp, err)
	}
}

func TestRefreshToken_PhotoPresignFailureDoesNotFailRefresh(t *testing.T) {
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	u.Photo = "me.png"
	e := newEnv(t, fakePhotoResolver{err: errors.New("storage down")}, u)
	pair := login(t, e, u.Phone)

	resp, err := e.uc.RefreshToken(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh must succeed without a photo URL: %v", err)
	}
	if resp.User.Photo != "" {
		t.Errorf("photo = %q, want empty", resp.User.Photo)
	}
}

// --- Logout --------------------------------------------------------------

func TestLogout_InvalidatesBothTokens(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	pair := login(t, e, u.Phone)

	if err := e.uc.Logout(ctx, pair.AccessToken, pair.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := e.uc.GetUserInfoFromToken(ctx, pair.AccessToken); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("access token after logout: err = %v, want ErrInvalidToken", err)
	}
	if _, err := e.uc.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("refresh token after logout: err = %v, want ErrInvalidToken", err)
	}
}

func TestLogout_RefreshTokenIsOptionalAndBestEffort(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)

	t.Run("without refresh token", func(t *testing.T) {
		e := newEnv(t, nil, u)
		pair := login(t, e, u.Phone)
		if err := e.uc.Logout(ctx, pair.AccessToken, ""); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if _, err := e.uc.GetUserInfoFromToken(ctx, pair.AccessToken); err == nil {
			t.Error("access token must be revoked")
		}
	})
	t.Run("garbage refresh token does not fail logout", func(t *testing.T) {
		e := newEnv(t, nil, u)
		pair := login(t, e, u.Phone)
		if err := e.uc.Logout(ctx, pair.AccessToken, "garbage"); err != nil {
			t.Fatalf("Logout: %v", err)
		}
	})
	t.Run("access token passed as refresh is not blacklisted twice or misused", func(t *testing.T) {
		e := newEnv(t, nil, u)
		pair := login(t, e, u.Phone)
		other := login(t, e, u.Phone)
		// other.AccessToken is of type access, so it must be ignored in the refresh slot.
		if err := e.uc.Logout(ctx, pair.AccessToken, other.AccessToken); err != nil {
			t.Fatalf("Logout: %v", err)
		}
		if _, err := e.uc.GetUserInfoFromToken(ctx, other.AccessToken); err != nil {
			t.Errorf("an unrelated access token must stay valid: %v", err)
		}
	})
}

func TestLogout_Rejects(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleUser, true)
	e := newEnv(t, nil, u)
	pair := login(t, e, u.Phone)

	if err := e.uc.Logout(ctx, pair.RefreshToken, ""); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("refresh token used as access: err = %v, want ErrInvalidToken", err)
	}
	if err := e.uc.Logout(ctx, "garbage", ""); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("garbage: err = %v, want ErrInvalidToken", err)
	}
}

// --- GetUserInfoFromToken ------------------------------------------------

func TestGetUserInfoFromToken(t *testing.T) {
	ctx := context.Background()
	u := newUser(t, "+996500500500", domainUser.RoleSuperuser, true)
	e := newEnv(t, nil, u)
	pair := login(t, e, u.Phone)

	got, err := e.uc.GetUserInfoFromToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("GetUserInfoFromToken: %v", err)
	}
	if got.ID != u.ID || got.Role != domainUser.RoleSuperuser || !got.IsActive {
		t.Errorf("unexpected user %+v", got)
	}

	if _, err := e.uc.GetUserInfoFromToken(ctx, pair.RefreshToken); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("a refresh token must not authenticate API calls: err = %v", err)
	}
	if _, err := e.uc.GetUserInfoFromToken(ctx, ""); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("empty token: err = %v, want ErrInvalidToken", err)
	}
}
