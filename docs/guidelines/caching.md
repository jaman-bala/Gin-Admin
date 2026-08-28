# Redis Caching Guidelines

Redis in this template currently backs two things: the token blacklist
(`internal/domain/token`) and the login rate limiter
(`middleware/ratelimit_redis.go`). `internal/application/analytics/usecase.go`
is the one place that caches actual data (dashboard user stats) — use it as
the reference implementation when caching anything else.

## What's worth caching

Split candidate data into two buckets before reaching for Redis:

- **Read-heavy, rarely-changing** — data that's requested on nearly every
  page load but only changes occasionally: dashboard aggregates, config-ish
  lookups, anything backed by a full-table scan or an expensive join. This
  is the `GetUserStats` case: a `COUNT(*) FILTER` aggregate over `users`,
  requested on every dashboard load/refresh.
- **Expensive to (re)compute, tolerant of staleness** — a generated report,
  a computed summary — where recomputing on every request is wasteful but a
  slightly stale answer is fine.

Data that's cheap to query directly (a single indexed row lookup) or that
must always be current (a permission check, a balance, anything
security-relevant) is **not** a caching candidate — caching it either buys
nothing or introduces a staleness bug.

## Pattern: cache-aside with a short TTL

Follow `analytics.usecase.GetUserStats` (`getCached` / `setCached`):

1. **Cache is optional at construction time.** `NewUseCase(repo, cache)`
   accepts `cache == nil` and every call falls straight through to the
   repository — this keeps tests simple and means a missing/misconfigured
   Redis doesn't change the usecase's public behavior, only its latency.
2. **Every Redis call gets its own short timeout**, independent of the
   caller's request context (`cacheCallTimeout = 150ms` in the analytics
   usecase). A stalled/partitioned Redis connection (not a fast
   "connection refused") must not add multi-second latency to a request
   that would otherwise complete fine from the database.
3. **A cache miss, a corrupt payload, and a Redis outage are the same
   case**: fall through to the database. Never let a cache-layer failure
   become a request failure — a slower response beats a broken one.
4. **Cache writes are best-effort.** If `Set` fails after a successful DB
   read, log and move on (`slog.Warn`) — don't fail the request over a
   failed cache write when you already have a good result to return.
5. **Pick the TTL from how stale an answer is acceptable**, not a
   one-size-fits-all default. The analytics cache uses 30s because a
   dashboard number that's up to 30 seconds old is a reasonable trade for
   not scanning `users` on every refresh; a different endpoint might
   tolerate seconds or minutes — write down *why* next to the constant, the
   way `userStatsCacheTTL`'s comment does.

## Interface shape

Define the cache dependency on the consumer side, scoped to what that
usecase actually needs — see `analytics.Cache` (`Get`/`Set` only) versus
the full `token.Repository` surface `redis.Cache` implements. This keeps
the application layer depending on a narrow shape it owns, not a concrete
Redis client, and `*redis.Cache` satisfies each narrower interface without
extra glue.

## What to avoid

- Don't cache anything a role/permission decision depends on — re-derive
  it from the verified token/DB on every request (see `security.md`).
- Don't introduce a cache without a fallback path. If a usecase can't
  function when Redis is down, caching was the wrong tool — fix the query
  or add a DB-side index instead (see `database.md`).
- Don't reach for a long TTL "to reduce load" without deciding what
  staleness window is actually acceptable to the feature — that decision
  belongs in a comment next to the constant, not left implicit.
