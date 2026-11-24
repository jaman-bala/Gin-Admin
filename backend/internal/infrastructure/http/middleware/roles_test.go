package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoleLevel(t *testing.T) {
	tests := []struct {
		role string
		want int
	}{
		{"superuser", RoleLevelSuperuser},
		{"admin", RoleLevelAdmin},
		{"user", RoleLevelUser},
		{"", RoleLevelNone},
		{"root", RoleLevelNone},
		{"Admin", RoleLevelNone}, // roles are case-sensitive; no accidental privilege via casing
		{"admin ", RoleLevelNone},
	}
	for _, tc := range tests {
		if got := roleLevel(tc.role); got != tc.want {
			t.Errorf("roleLevel(%q) = %d, want %d", tc.role, got, tc.want)
		}
	}
	if RoleLevelUser >= RoleLevelAdmin || RoleLevelAdmin >= RoleLevelSuperuser {
		t.Error("role levels must be strictly ordered user < admin < superuser")
	}
}

func TestRequireRoleLevel_MissingRoleInContext(t *testing.T) {
	// No AuthMiddleware in front: the role key was never set.
	w, reached, _ := do(t, "", RequireRoleLevelMiddleware(RoleLevelUser))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if *reached {
		t.Error("handler must not run when the role is missing")
	}
	if !strings.Contains(w.Body.String(), "AUTH_ROLE_MISSING") {
		t.Errorf("body = %s, want AUTH_ROLE_MISSING", w.Body.String())
	}
}

func TestRequireRoleLevel_ForbiddenBody(t *testing.T) {
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) { c.Set("role", "user") },
		RequireRoleLevelMiddleware(RoleLevelAdmin),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "AUTH_INSUFFICIENT_PRIVILEGES") {
		t.Errorf("body = %s, want AUTH_INSUFFICIENT_PRIVILEGES", w.Body.String())
	}
}
