package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"gin_auth_service/internal/domain/auditlog"

	"github.com/gin-gonic/gin"
)

type captureRecorder struct{ got []*auditlog.AuditLog }

func (c *captureRecorder) Record(e *auditlog.AuditLog) { c.got = append(c.got, e) }

// audited sends one request through RequestID + AuditMiddleware and returns
// the single entry that was recorded.
func audited(t *testing.T, method, target string, hdr map[string]string) *auditlog.AuditLog {
	t.Helper()
	rec := &captureRecorder{}
	r := gin.New()
	r.Use(RequestIDMiddleware(), AuditMiddleware(rec))
	r.Any("/*path", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
	if len(rec.got) != 1 {
		t.Fatalf("recorded %d entries, want 1", len(rec.got))
	}
	return rec.got[0]
}

// audit_logs.user_agent is VARCHAR(255) (counted in characters): anything
// longer makes PostgreSQL reject the INSERT.
const userAgentColumnLimit = 255

func TestAudit_UserAgentFitsTheColumn(t *testing.T) {
	tests := []struct {
		name string
		ua   string
	}{
		{"long ascii", strings.Repeat("X", 5000)},
		{"long multibyte (bytes > 255 well before runes)", strings.Repeat("ж", 400)},
		{"exactly at the limit stays intact", strings.Repeat("a", userAgentColumnLimit)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := audited(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"User-Agent": tc.ua})
			if n := utf8.RuneCountInString(e.UserAgent); n > userAgentColumnLimit {
				t.Errorf("user agent has %d chars, column holds %d", n, userAgentColumnLimit)
			}
			if !utf8.ValidString(e.UserAgent) {
				t.Error("truncation must not cut a multibyte character in half")
			}
		})
	}
	e := audited(t, http.MethodGet, "/x", map[string]string{"User-Agent": strings.Repeat("a", userAgentColumnLimit)})
	if utf8.RuneCountInString(e.UserAgent) != userAgentColumnLimit {
		t.Error("a user agent that fits must be stored unchanged")
	}
}

// PostgreSQL rejects invalid UTF-8 and NUL bytes in text columns.
func TestAudit_UntrustedTextIsSafeForPostgres(t *testing.T) {
	e := audited(t, http.MethodGet, "/x/%00y", map[string]string{"User-Agent": "bad\xff\xfebytes"})

	for name, v := range map[string]string{"user agent": e.UserAgent, "data": e.Data} {
		if !utf8.ValidString(v) {
			t.Errorf("%s is not valid UTF-8: %q", name, v)
		}
		if strings.ContainsRune(v, 0) {
			t.Errorf("%s contains a NUL byte: %q", name, v)
		}
	}
}

func TestAudit_DataIsBounded(t *testing.T) {
	e := audited(t, http.MethodGet, "/"+strings.Repeat("a", 20000), nil)
	if n := utf8.RuneCountInString(e.Data); n > 1024 {
		t.Errorf("data has %d chars; an unauthenticated client must not be able to store arbitrary volumes", n)
	}
}

func TestAudit_ClientSuppliedRequestIDCannotInflateTheEntry(t *testing.T) {
	e := audited(t, http.MethodPost, "/api/v1/auth/login", map[string]string{RequestIDHeader: strings.Repeat("Z", 100000)})
	if len(e.Data) > 1024 {
		t.Errorf("data is %d bytes long", len(e.Data))
	}
}
