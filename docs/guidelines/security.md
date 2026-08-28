# Security Checklist

This template already implements the baseline protections listed below. When
you extend it — a new handler, a new usecase, a new field on an existing
entity — check the relevant section before merging.

## Input validation

Every value that *can* be validated before it reaches business logic
*should* be — reject bad input at the edge instead of letting it flow into a
usecase or a query.

- New request DTOs go through `binding:"required"` / `validate:"..."` tags
  (`internal/pkg/validator`). If a field has a real shape constraint (phone
  format, enum of allowed values, length bounds), encode it in the tag —
  don't rely on the handler or usecase to notice.
- `strong_password` (`internal/pkg/validator/custom.go`) is the pattern for
  a custom rule: register it once in `InitCustomValidators`, reuse the tag
  everywhere a password is accepted.
- `BodySizeLimit` (`internal/infrastructure/http/middleware/bodylimit.go`) is
  applied globally in `router.go` — a new route doesn't need its own limit,
  but if it accepts unusually large payloads (bulk import, file upload),
  size it explicitly rather than inheriting the global default silently.
- Never build a SQL string by concatenating request input — every
  `postgres/*_repository.go` query uses `sqlx` bind parameters (`$1`, `$2`,
  ...). Keep it that way for any new query.
- Treat `deleted_at IS NULL` / role checks the same way: a query that reads
  or writes a row must filter by the invariants that make the row valid to
  touch (soft-delete, ownership, role), not assume the caller already did.

## Concurrency

- If a table can be edited by two admins at once, use the optimistic-lock
  pattern from `users.version` (migration `00007`): bump `version` on every
  `UPDATE`, check it in the `WHERE` clause, and surface `ErrStaleWrite` (see
  `pkg/errors/errors.go`) as `409 Conflict` instead of silently dropping one
  writer's change. Don't add "last write wins" mutation paths to entities
  that are edited by more than one actor.
- Rate limiting and blacklist counters go through the Lua script in
  `internal/infrastructure/redis/cache.go` (`incrementScript`) — a bare
  `INCR` followed by a separate `EXPIRE` is two round trips and a race
  window between them. Reuse the same atomic-script pattern for any new
  Redis counter.

## Authentication

- Tokens are accepted from the `Authorization: Bearer <token>` header only
  (`middleware/auth.go`). Don't add cookie-based auth without adding CSRF
  protection alongside it — the comment in `auth.go` documents why cookies
  were skipped; that tradeoff needs to be revisited explicitly, not
  bypassed silently.
- Authentication and authorization decisions live server-side only
  (`AuthMiddleware`, `RequireRoleLevelMiddleware`). Never trust a role or
  user ID supplied by the client in a request body or query param — always
  read it from the verified token (`c.Get("id")`, `c.Get("role")`) that
  `AuthMiddleware` set.
- `LoginRateLimitMiddleware` limits brute-force attempts (5 / 15 min per
  IP) and fails **open** on Redis outage — a cache failure must not lock out
  legitimate users. Apply the same fail-open principle to any new
  Redis-backed security check: an infra outage should degrade gracefully,
  not turn into a denial of service against your own users.
- Passwords are bcrypt-hashed (`internal/pkg/hash`) and never logged or
  returned in a response DTO. Grep any new DTO's `json:` tags before adding
  a field to make sure nothing sensitive leaks into a response by accident.

## Authorization

- Role checks belong in `RequireRoleLevelMiddleware(minLevel)` at the route
  group level (`router.go`), not scattered inline in handlers. Adding a new
  role-gated route means picking the right `RoleLevel*` constant, not
  writing a new ad-hoc `if role != "admin"` check.
- Minimize granted privileges: a new endpoint should default to the
  narrowest role level that can still do the job, not `admin` "to be safe"
  and not `user` "to avoid friction."

## Error responses

- Never echo a raw `err.Error()` from an infrastructure dependency (DB
  driver, Redis client, JWT parser) back to the client — it can leak
  internal details. `middleware/auth.go` and `handler/errors.go` already
  follow this: infra errors get logged server-side (`slog.Error`) and
  mapped to a generic client-facing message. Follow `respondError`'s
  pattern (`handler/errors.go`) — map known domain sentinel errors
  (`pkg/errors/errors.go`) to the right HTTP status, and let anything
  unrecognized fall through to a logged `500` with a generic body.

## Secrets

- Secrets are read via `config.readSecret()`: Docker secret file at
  `/run/secrets/<lowercase_key>` first, then the environment variable, then
  a fallback. Any new secret (API key, third-party credential) should go
  through this same helper — not a bare `os.Getenv`, which skips the
  Swarm-secrets path and encourages committing a fallback value that looks
  like a real credential.
- Don't hardcode a non-empty fallback for anything that is actually a
  secret (see `SECRET_KEY`, `DB_PASSWORD`, `MINIO_SECRET_KEY` — all default
  to `""`, forcing an explicit value in every environment).

## Client-side trust

- CORS is allow-listed by origin (`CORS_ALLOWED_ORIGINS`) — extend the list
  per environment, never `AllowOrigins: []string{"*"}` on a route that
  accepts credentials.
- The frontend must never be the sole enforcer of a permission — a
  disabled button in the React app is a UX nicety, not a security boundary.
  Every permission it reflects must already be enforced by
  `RequireRoleLevelMiddleware` on the backend route.
