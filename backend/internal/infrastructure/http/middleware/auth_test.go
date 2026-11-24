package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gin_auth_service/internal/application/auth"
	domainUser "gin_auth_service/internal/domain/user"

	"github.com/gin-gonic/gin"
	"uuid"
)

func init() { gin.SetMode(gin.TestMode) }

// stubAuth implements auth.UseCase; only GetUserInfoFromToken is exercised.
type stubAuth struct {
	auth.UseCase
	user      *domainUser.User
	err       error
	gotToken  string
	callCount int
}

func (s *stubAuth) GetUserInfoFromToken(_ context.Context, token string) (*domainUser.User, error) {
	s.callCount++
	s.gotToken = token
	return s.user, s.err
}

// do runs one request through the given middlewares and a terminal handler
// that records whether it was reached and what the middleware put in context.
func do(t *testing.T, header string, mws ...gin.HandlerFunc) (*httptest.ResponseRecorder, *bool, *gin.Context) {
	t.Helper()
	reached := false
	var ctxSeen *gin.Context

	r := gin.New()
	r.GET("/x", append(mws, func(c *gin.Context) {
		reached = true
		ctxSeen = c
		c.Status(http.StatusOK)
	})...)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, &reached, ctxSeen
}

func TestAuthMiddleware_RejectsMissingOrMalformedHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no header", ""},
		{"empty bearer", "Bearer "},
		{"basic scheme", "Basic dXNlcjpwYXNz"},
		{"lowercase bearer", "bearer abc"},
		{"bare token without scheme", "abc.def.ghi"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAuth{}
			w, reached, _ := do(t, tc.header, AuthMiddleware(stub))

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
			if *reached {
				t.Error("handler must not run without a valid token")
			}
			if stub.callCount != 0 {
				t.Error("use case must not be consulted for a header that carries no token")
			}
			if !strings.Contains(w.Body.String(), "AUTH_TOKEN_MISSING") {
				t.Errorf("body = %s, want AUTH_TOKEN_MISSING", w.Body.String())
			}
		})
	}
}

func TestAuthMiddleware_InvalidTokenDoesNotLeakInternals(t *testing.T) {
	stub := &stubAuth{err: errors.New("redis: dial tcp 10.0.0.5:6379: connection refused")}
	w, reached, _ := do(t, "Bearer some.jwt.token", AuthMiddleware(stub))

	if w.Code != http.StatusUnauthorized || *reached {
		t.Fatalf("status = %d reached = %v, want 401 and not reached", w.Code, *reached)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AUTH_TOKEN_INVALID") {
		t.Errorf("body = %s, want AUTH_TOKEN_INVALID", body)
	}
	for _, leak := range []string{"redis", "10.0.0.5", "connection refused"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks internal detail %q: %s", leak, body)
		}
	}
	if stub.gotToken != "some.jwt.token" {
		t.Errorf("token passed to use case = %q, want the bearer value without the prefix", stub.gotToken)
	}
}

func TestAuthMiddleware_ValidTokenPopulatesContext(t *testing.T) {
	u := &domainUser.User{ID: uuid.NewV7(), Role: domainUser.RoleAdmin, IsActive: true}
	w, reached, c := do(t, "Bearer good", AuthMiddleware(&stubAuth{user: u}))

	if w.Code != http.StatusOK || !*reached {
		t.Fatalf("status = %d reached = %v, want 200 and reached", w.Code, *reached)
	}
	if got, _ := c.Get("id"); got != u.ID {
		t.Errorf("ctx id = %v, want %v", got, u.ID)
	}
	if got, _ := c.Get("role"); got != "admin" {
		t.Errorf("ctx role = %v, want admin", got)
	}
	if got, _ := c.Get("user"); got != u {
		t.Errorf("ctx user = %v, want %v", got, u)
	}
}

// TestAuthThenRole_Chain checks the real wiring used by the router:
// AuthMiddleware feeds RequireRoleLevelMiddleware through the "role" context key.
func TestAuthThenRole_Chain(t *testing.T) {
	tests := []struct {
		name     string
		role     domainUser.Role
		minLevel int
		want     int
	}{
		{"user on user route", domainUser.RoleUser, RoleLevelUser, http.StatusOK},
		{"user on admin route", domainUser.RoleUser, RoleLevelAdmin, http.StatusForbidden},
		{"admin on admin route", domainUser.RoleAdmin, RoleLevelAdmin, http.StatusOK},
		{"admin on superuser route", domainUser.RoleAdmin, RoleLevelSuperuser, http.StatusForbidden},
		{"superuser on admin route", domainUser.RoleSuperuser, RoleLevelAdmin, http.StatusOK},
		{"token with unknown role on user route", domainUser.Role("root"), RoleLevelUser, http.StatusForbidden},
		{"token with empty role on user route", domainUser.Role(""), RoleLevelUser, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAuth{user: &domainUser.User{ID: uuid.NewV7(), Role: tc.role, IsActive: true}}
			w, reached, _ := do(t, "Bearer t", AuthMiddleware(stub), RequireRoleLevelMiddleware(tc.minLevel))

			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tc.want, w.Body.String())
			}
			if *reached != (tc.want == http.StatusOK) {
				t.Errorf("handler reached = %v, want %v", *reached, tc.want == http.StatusOK)
			}
		})
	}
}
