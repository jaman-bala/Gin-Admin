package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func requestIDFor(t *testing.T, incoming string) string {
	t.Helper()
	r := gin.New()
	r.Use(RequestIDMiddleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if incoming != "" {
		req.Header.Set(RequestIDHeader, incoming)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Header().Get(RequestIDHeader)
}

func TestRequestID_AcceptsSaneClientValue(t *testing.T) {
	for _, id := range []string{"abc-123", "trace.id_42", "0198f6c2-7b1e-7a2b-8c3d-1234567890ab"} {
		if got := requestIDFor(t, id); got != id {
			t.Errorf("request id %q was replaced with %q", id, got)
		}
	}
}

func TestRequestID_ReplacesUnsafeClientValue(t *testing.T) {
	tests := map[string]string{
		"too long":        strings.Repeat("a", 65),
		"spaces":          "a b",
		"newline":         "a\nb",
		"log forging":     "x level=ERROR msg=pwned",
		"non ascii":       "идентификатор",
		"path separators": "../../etc/passwd",
	}
	for name, bad := range tests {
		t.Run(name, func(t *testing.T) {
			got := requestIDFor(t, bad)
			if got == bad || got == "" {
				t.Fatalf("unsafe request id %q was kept or dropped: %q", bad, got)
			}
			if len(got) > 64 {
				t.Errorf("generated id is too long: %q", got)
			}
		})
	}
}

func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	a, b := requestIDFor(t, ""), requestIDFor(t, "")
	if a == "" || a == b {
		t.Errorf("want two distinct generated ids, got %q and %q", a, b)
	}
}
