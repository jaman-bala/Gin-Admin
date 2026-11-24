package token

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gin_auth_service/internal/pkg/jwt"
	appErrors "gin_auth_service/pkg/errors"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const testSecret = "unit-test-secret-key-0123456789abcdef"

// memRepo is an in-memory domain/token.Repository that records the TTL of
// every key so tests can assert blacklist lifetimes.
type memRepo struct {
	mu      sync.Mutex
	keys    map[string]time.Duration
	setErr  error
	existsE error
}

func newMemRepo() *memRepo { return &memRepo{keys: map[string]time.Duration{}} }

func (r *memRepo) Set(_ context.Context, key string, _ any, ttl time.Duration) error {
	if r.setErr != nil {
		return r.setErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key] = ttl
	return nil
}

func (r *memRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.keys, key)
	return nil
}

func (r *memRepo) Exists(_ context.Context, key string) (bool, error) {
	if r.existsE != nil {
		return false, r.existsE
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.keys[key]
	return ok, nil
}

func newUC(repo *memRepo) UseCase {
	return NewUseCase(repo, jwt.NewJWTService(testSecret))
}

func mustPair(t *testing.T, uc UseCase, access, refresh time.Duration) *TokenPair {
	t.Helper()
	pair, err := uc.GenerateTokenPair(context.Background(), "11111111-1111-7111-8111-111111111111", "admin", true, access, refresh)
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	return pair
}

func TestGenerateTokenPair_ClaimsRoundTrip(t *testing.T) {
	uc := newUC(newMemRepo())
	pair := mustPair(t, uc, 15*time.Minute, time.Hour)

	access, err := uc.GetTokenInfo(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("GetTokenInfo(access): %v", err)
	}
	if access.Type != "access" || access.Role != "admin" || !access.IsActive || access.UserID != "11111111-1111-7111-8111-111111111111" {
		t.Errorf("unexpected access claims: %+v", access)
	}

	refresh, err := uc.GetTokenInfo(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("GetTokenInfo(refresh): %v", err)
	}
	if refresh.Type != "refresh" {
		t.Errorf("refresh token type = %q, want refresh", refresh.Type)
	}
	if access.JTI == refresh.JTI {
		t.Error("access and refresh tokens must have distinct jti, otherwise blacklisting one kills the other")
	}
	if !refresh.ExpiresAt.After(access.ExpiresAt) {
		t.Error("refresh token must outlive the access token")
	}
}

func TestGetTokenInfo_Rejects(t *testing.T) {
	ctx := context.Background()
	uc := newUC(newMemRepo())
	valid := mustPair(t, uc, time.Minute, time.Hour).AccessToken

	sign := func(method jwtlib.SigningMethod, key any, claims jwtlib.MapClaims) string {
		s, err := jwtlib.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return s
	}
	future := time.Now().Add(time.Hour).Unix()

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"tampered payload", valid[:len(valid)-4] + "AAAA"},
		{"signed with another secret", sign(jwtlib.SigningMethodHS256, []byte("another-secret-another-secret-123456"),
			jwtlib.MapClaims{"jti": "x", "exp": future, "type": "access"})},
		{"expired", sign(jwtlib.SigningMethodHS256, []byte(testSecret),
			jwtlib.MapClaims{"jti": "x", "exp": time.Now().Add(-time.Minute).Unix(), "type": "access"})},
		{"missing jti", sign(jwtlib.SigningMethodHS256, []byte(testSecret),
			jwtlib.MapClaims{"exp": future, "type": "access"})},
		{"alg none", sign(jwtlib.SigningMethodNone, jwtlib.UnsafeAllowNoneSignatureType,
			jwtlib.MapClaims{"jti": "x", "exp": future, "type": "access"})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := uc.GetTokenInfo(ctx, tc.token)
			if !errors.Is(err, appErrors.ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
			if info != nil {
				t.Errorf("info must be nil on error, got %+v", info)
			}
		})
	}
}

func TestBlacklistToken_RevokesToken(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepo()
	uc := newUC(repo)
	pair := mustPair(t, uc, time.Minute, time.Hour)

	info, err := uc.GetTokenInfo(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("GetTokenInfo before blacklist: %v", err)
	}
	if err := uc.BlacklistToken(ctx, pair.AccessToken, info.ExpiresAt); err != nil {
		t.Fatalf("BlacklistToken: %v", err)
	}

	if _, err := uc.GetTokenInfo(ctx, pair.AccessToken); !errors.Is(err, appErrors.ErrInvalidToken) {
		t.Errorf("blacklisted access token: err = %v, want ErrInvalidToken", err)
	}
	if _, err := uc.GetTokenInfo(ctx, pair.RefreshToken); err != nil {
		t.Errorf("blacklisting the access token must not revoke the refresh token: %v", err)
	}

	blacklisted, err := uc.IsTokenBlacklisted(ctx, pair.AccessToken)
	if err != nil || !blacklisted {
		t.Errorf("IsTokenBlacklisted = %v, %v; want true, nil", blacklisted, err)
	}
}

func TestBlacklistToken_TTL(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepo()
	uc := newUC(repo)
	pair := mustPair(t, uc, time.Minute, time.Hour)

	t.Run("future expiry keeps the remaining lifetime", func(t *testing.T) {
		if err := uc.BlacklistToken(ctx, pair.AccessToken, time.Now().Add(10*time.Minute)); err != nil {
			t.Fatal(err)
		}
		for _, ttl := range repo.keys {
			if ttl <= 9*time.Minute || ttl > 10*time.Minute {
				t.Errorf("ttl = %v, want ~10m", ttl)
			}
		}
	})

	t.Run("past expiry falls back to a positive ttl", func(t *testing.T) {
		repo2 := newMemRepo()
		uc2 := newUC(repo2)
		p := mustPair(t, uc2, time.Minute, time.Hour)
		if err := uc2.BlacklistToken(ctx, p.RefreshToken, time.Now().Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
		for _, ttl := range repo2.keys {
			if ttl <= 0 {
				t.Errorf("ttl = %v; a non-positive ttl would mean 'no expiry' or an immediate drop in Redis", ttl)
			}
		}
	})
}

func TestBlacklistToken_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("empty token", func(t *testing.T) {
		if err := newUC(newMemRepo()).BlacklistToken(ctx, "", time.Now()); !errors.Is(err, appErrors.ErrInvalidToken) {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})
	t.Run("malformed token", func(t *testing.T) {
		if err := newUC(newMemRepo()).BlacklistToken(ctx, "garbage", time.Now()); !errors.Is(err, appErrors.ErrInvalidToken) {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})
	t.Run("storage failure is propagated", func(t *testing.T) {
		repo := newMemRepo()
		uc := newUC(repo)
		pair := mustPair(t, uc, time.Minute, time.Hour)
		boom := errors.New("redis down")
		repo.setErr = boom
		if err := uc.BlacklistToken(ctx, pair.AccessToken, time.Now().Add(time.Minute)); !errors.Is(err, boom) {
			t.Errorf("err = %v, want the storage error", err)
		}
	})
}

func TestGetTokenInfo_BlacklistLookupFailureIsNotSilentlyAllowed(t *testing.T) {
	repo := newMemRepo()
	uc := newUC(repo)
	pair := mustPair(t, uc, time.Minute, time.Hour)

	boom := errors.New("redis down")
	repo.existsE = boom
	info, err := uc.GetTokenInfo(context.Background(), pair.AccessToken)
	if err == nil || info != nil {
		t.Fatalf("got info=%+v err=%v; a Redis outage must fail closed, not accept the token", info, err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap the storage error", err)
	}
	if errors.Is(err, appErrors.ErrInvalidToken) {
		t.Error("an infrastructure failure must not be reported as an invalid token (client would see 401 instead of 5xx)")
	}
}

func TestIsTokenBlacklisted_EdgeCases(t *testing.T) {
	ctx := context.Background()
	uc := newUC(newMemRepo())

	if _, err := uc.IsTokenBlacklisted(ctx, ""); err == nil {
		t.Error("empty token must return an error")
	}
	if got, err := uc.IsTokenBlacklisted(ctx, "garbage"); err != nil || got {
		t.Errorf("malformed token: got %v, %v; want false, nil (rejected later by signature check)", got, err)
	}
}
