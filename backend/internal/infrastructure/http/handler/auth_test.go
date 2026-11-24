package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gin_auth_service/internal/application/auth"
	appValidator "gin_auth_service/internal/pkg/validator"
	appErrors "gin_auth_service/pkg/errors"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
	// Same call main.go makes: switches Gin's validator to the `validate` tag.
	// Without it the DTO rules below would silently not be enforced, so these
	// tests also guard the validator wiring across library upgrades.
	appValidator.InitCustomValidators()
}

// stubAuth implements auth.UseCase with per-method hooks.
type stubAuth struct {
	auth.UseCase
	login   func(auth.LoginRequestDTO) (*auth.LoginResponseDTO, error)
	refresh func(string) (*auth.TokenResponseDTO, error)
	logout  func(access, refresh string) error
	calls   int
}

func (s *stubAuth) Login(_ context.Context, req auth.LoginRequestDTO) (*auth.LoginResponseDTO, error) {
	s.calls++
	return s.login(req)
}
func (s *stubAuth) RefreshToken(_ context.Context, rt string) (*auth.TokenResponseDTO, error) {
	s.calls++
	return s.refresh(rt)
}
func (s *stubAuth) Logout(_ context.Context, a, r string) error {
	s.calls++
	return s.logout(a, r)
}

func newRouter(h *AuthHandler) *gin.Engine {
	r := gin.New()
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	return r
}

func post(r *gin.Engine, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func jsonField(t *testing.T, w *httptest.ResponseRecorder, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not JSON: %v (%s)", err, w.Body.String())
	}
	s, _ := m[key].(string)
	return s
}

// --- Login: request validation ------------------------------------------

func TestLogin_RejectsInvalidBodyBeforeCallingUseCase(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ``},
		{"malformed json", `{"phone": `},
		{"empty object", `{}`},
		{"missing password", `{"phone":"+996500500500"}`},
		{"missing phone", `{"password":"Password123"}`},
		{"phone not e164", `{"phone":"not-a-phone","password":"Password123"}`},
		{"phone too short", `{"phone":"+99650","password":"Password123"}`},
		{"wrong field types", `{"phone":123,"password":true}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAuth{}
			w := post(newRouter(NewAuthHandler(stub)), "/login", tc.body, nil)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
			if stub.calls != 0 {
				t.Error("use case must not be called with an invalid request")
			}
			if jsonField(t, w, "error") == "" {
				t.Errorf("body must carry an error message: %s", w.Body.String())
			}
		})
	}
}

// validator/v10 >= 10.30 treats the leading "+" in `e164` as optional (it was
// required in 10.26). That is safe only because every use case runs
// utils.NormalizePhone after binding, so the stored value always has the "+".
// This test pins the contract: the handler forwards the phone untouched and
// normalization stays the use case's job.
func TestLogin_PhoneWithoutPlusPassesValidationAndIsNormalizedByUseCase(t *testing.T) {
	var got string
	stub := &stubAuth{login: func(req auth.LoginRequestDTO) (*auth.LoginResponseDTO, error) {
		got = req.Phone
		return &auth.LoginResponseDTO{}, nil
	}}
	w := post(newRouter(NewAuthHandler(stub)), "/login", `{"phone":"996500500500","password":"Password123"}`, nil)

	if w.Code != http.StatusOK || got != "996500500500" {
		t.Fatalf("status = %d phone = %q; want 200 and the raw phone forwarded", w.Code, got)
	}
}

// --- Login: error mapping ------------------------------------------------

func TestLogin_MapsUseCaseResults(t *testing.T) {
	const body = `{"phone":"+996500500500","password":"Password123"}`

	t.Run("success", func(t *testing.T) {
		stub := &stubAuth{login: func(req auth.LoginRequestDTO) (*auth.LoginResponseDTO, error) {
			if req.Phone != "+996500500500" || req.Password != "Password123" {
				t.Errorf("use case got %+v", req)
			}
			return &auth.LoginResponseDTO{AccessToken: "a", RefreshToken: "r", Message: "ok"}, nil
		}}
		w := post(newRouter(NewAuthHandler(stub)), "/login", body, nil)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if jsonField(t, w, "access_token") != "a" || jsonField(t, w, "refresh_token") != "r" {
			t.Errorf("unexpected body %s", w.Body.String())
		}
	})

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"wrong credentials", appErrors.ErrInvalidCredentials, http.StatusUnauthorized},
		{"blocked", appErrors.ErrAccountBlocked, http.StatusForbidden},
		{"wrapped sentinel is still mapped", fmt.Errorf("login: %w", appErrors.ErrInvalidCredentials), http.StatusUnauthorized},
		{"unexpected error", errors.New("pq: connection reset"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAuth{login: func(auth.LoginRequestDTO) (*auth.LoginResponseDTO, error) { return nil, tc.err }}
			w := post(newRouter(NewAuthHandler(stub)), "/login", body, nil)

			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

// --- Refresh -------------------------------------------------------------

func TestRefresh(t *testing.T) {
	t.Run("malformed json", func(t *testing.T) {
		stub := &stubAuth{}
		w := post(newRouter(NewAuthHandler(stub)), "/refresh", `{`, nil)
		if w.Code != http.StatusBadRequest || stub.calls != 0 {
			t.Errorf("status = %d calls = %d, want 400 and 0", w.Code, stub.calls)
		}
	})
	t.Run("success forwards the token", func(t *testing.T) {
		stub := &stubAuth{refresh: func(rt string) (*auth.TokenResponseDTO, error) {
			if rt != "old-refresh" {
				t.Errorf("use case got %q", rt)
			}
			return &auth.TokenResponseDTO{AccessToken: "a2", RefreshToken: "r2"}, nil
		}}
		w := post(newRouter(NewAuthHandler(stub)), "/refresh", `{"refresh_token":"old-refresh"}`, nil)
		if w.Code != http.StatusOK || jsonField(t, w, "access_token") != "a2" {
			t.Errorf("status = %d body = %s", w.Code, w.Body.String())
		}
	})
	t.Run("invalid token is 401", func(t *testing.T) {
		stub := &stubAuth{refresh: func(string) (*auth.TokenResponseDTO, error) { return nil, appErrors.ErrInvalidToken }}
		w := post(newRouter(NewAuthHandler(stub)), "/refresh", `{"refresh_token":"x"}`, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
	t.Run("blocked user is 403", func(t *testing.T) {
		stub := &stubAuth{refresh: func(string) (*auth.TokenResponseDTO, error) { return nil, appErrors.ErrAccountBlocked }}
		w := post(newRouter(NewAuthHandler(stub)), "/refresh", `{"refresh_token":"x"}`, nil)
		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})
}

// --- Logout --------------------------------------------------------------

func TestLogout(t *testing.T) {
	t.Run("missing authorization header", func(t *testing.T) {
		stub := &stubAuth{}
		w := post(newRouter(NewAuthHandler(stub)), "/logout", ``, nil)
		if w.Code != http.StatusUnauthorized || stub.calls != 0 {
			t.Errorf("status = %d calls = %d, want 401 and 0", w.Code, stub.calls)
		}
	})
	t.Run("success with optional refresh token", func(t *testing.T) {
		stub := &stubAuth{logout: func(access, refresh string) error {
			if access != "acc" || refresh != "ref" {
				t.Errorf("use case got access=%q refresh=%q", access, refresh)
			}
			return nil
		}}
		w := post(newRouter(NewAuthHandler(stub)), "/logout", `{"refresh_token":"ref"}`,
			map[string]string{"Authorization": "Bearer acc"})
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
	t.Run("success without body", func(t *testing.T) {
		stub := &stubAuth{logout: func(access, refresh string) error {
			if access != "acc" || refresh != "" {
				t.Errorf("use case got access=%q refresh=%q", access, refresh)
			}
			return nil
		}}
		w := post(newRouter(NewAuthHandler(stub)), "/logout", ``, map[string]string{"Authorization": "Bearer acc"})
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
	t.Run("invalid token is 401", func(t *testing.T) {
		stub := &stubAuth{logout: func(string, string) error { return appErrors.ErrInvalidToken }}
		w := post(newRouter(NewAuthHandler(stub)), "/logout", ``, map[string]string{"Authorization": "Bearer bad"})
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
}

// --- respondError --------------------------------------------------------

func TestRespondError_StatusMapping(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{appErrors.ErrUserNotFound, http.StatusNotFound},
		{appErrors.ErrConflict, http.StatusConflict},
		{appErrors.ErrPhoneAlreadyExists, http.StatusConflict},
		{appErrors.ErrStaleWrite, http.StatusConflict},
		{appErrors.ErrInvalidCredentials, http.StatusUnauthorized},
		{appErrors.ErrInvalidToken, http.StatusUnauthorized},
		{appErrors.ErrAccountBlocked, http.StatusForbidden},
		{appErrors.ErrForbiddenRole, http.StatusForbidden},
		{appErrors.ErrLastSuperuser, http.StatusConflict},
		{appErrors.ErrInvalidUUID, http.StatusBadRequest},
		{appErrors.ErrInvalidPatchField, http.StatusBadRequest},
		{fmt.Errorf("ctx: %w", appErrors.ErrUserNotFound), http.StatusNotFound},
		{errors.New("anything else"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.err.Error(), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

			respondError(c, tc.err)

			if w.Code != tc.want {
				t.Errorf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}

func TestRespondError_InternalErrorsAreNotExposed(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	respondError(c, errors.New(`pq: password authentication failed for user "gin_admin" at 10.0.0.7`))

	body := w.Body.String()
	for _, leak := range []string{"pq:", "gin_admin", "10.0.0.7"} {
		if strings.Contains(body, leak) {
			t.Errorf("500 response leaks %q: %s", leak, body)
		}
	}
	if jsonField(t, w, "error") != "internal server error" {
		t.Errorf("body = %s", body)
	}
}
