# PostgreSQL: Indexes & Transactions

Guidelines for writing new migrations (`backend/migrations/`, goose) and
queries (`backend/internal/infrastructure/postgres/`) in this template.
Grounded in the patterns already in the migration history — extend them the
same way rather than inventing a new one per feature.

## Indexes

Add an index when a query filters, sorts, or joins on a column at scale —
not preemptively on every column. Each existing index maps to one real
access pattern; use the same reasoning for a new one:

| Pattern | Example in this repo | When to reach for it |
|---|---|---|
| Partial unique index | `ux_users_phone_active` (`00005`) — unique on `phone` `WHERE deleted_at IS NULL` | A uniqueness constraint that should only apply to "live" rows, so a soft-deleted row doesn't block re-registration |
| Trigram GIN index | `ix_users_first_name_trgm` etc. (`00005`) | Backing an `ILIKE '%term%'` search — a plain B-tree index can't serve a leading-wildcard match |
| Partial covering index for a listing | `ix_users_live_created_at` (`00006`) | The default listing query does `WHERE deleted_at IS NULL ORDER BY created_at DESC` — the index mirrors the exact filter + sort, not just the sort column alone |
| Plain index for a filter/FK column | `ix_audit_logs_user_id`, `ix_audit_logs_entity_id` (`00006`) | Any column a handler filters or joins on regularly (an audit log lookup by user or entity) |

Before adding an index to a new table:
1. Write the actual query first (in the repository, or a scratch `EXPLAIN
   ANALYZE`), then index what it filters/sorts on — don't guess.
2. If the query only ever touches live (non-deleted) rows, make it a
   **partial** index (`WHERE deleted_at IS NULL`) rather than a full one —
   smaller index, same benefit, no dead-row bloat.
3. Every `Up` migration that adds an index needs the matching `Down` that
   drops it (see any migration's `-- +goose Down` block) — goose migrations
   in this repo are always reversible.

## Referential integrity

`audit_logs.user_id → users.id` uses `ON DELETE SET NULL` (`00006`), because
the application only ever soft-deletes (`users.deleted_at`), but the FK
still protects against orphaned/typo'd IDs if a row is ever hard-deleted
outside the app. When adding a new FK relationship:
- Decide the `ON DELETE` behavior deliberately (`SET NULL` to preserve
  history, `CASCADE` to remove dependents, `RESTRICT` to forbid the delete)
  — don't leave it as the implicit default.
- If the referencing table records history (audit-log-shaped), prefer `SET
  NULL` over `CASCADE` so history survives.

## Concurrency: optimistic locking

`users.version` (`00007`) is the template's pattern for "two admins editing
the same row at once shouldn't silently drop one writer's change":

```sql
UPDATE users SET ..., version = version + 1
WHERE id = $1 AND version = $2
```

If the `UPDATE` affects zero rows, the version the caller had was stale —
map that to `ErrStaleWrite` → `409 Conflict` (see `handler/errors.go`),
never to a swallowed no-op. Use this pattern for any new entity that's
edited through a read-then-write flow by more than one actor (not for
append-only or single-owner tables, where it's unneeded overhead).

## Transaction isolation

Postgres defaults to **Read Committed**, which is the right choice for
nearly everything in this template — most usecases do a single
read-modify-write on one row, and `users.version` already covers the lost-
update case above without needing a stricter isolation level.

Reach for something stronger only when you have a concrete anomaly to
prevent, not by default:

- **Repeatable Read** — a usecase runs multiple queries in one transaction
  and needs them all to see the same snapshot (e.g., an aggregate report
  that reads several tables and must not see a row appear between two of
  its queries).
- **Serializable** — a transaction does a "check invariant, then write"
  step where a concurrent transaction doing the same check could both pass
  and both write, breaking the invariant (classic case: two transfers that
  both check "balance >= amount" against the same starting balance). This
  template doesn't currently have that shape of usecase; if you add one
  (e.g., a balance/ledger feature), wrap it in `SERIALIZABLE` and handle the
  serialization-failure error with a retry, rather than reaching for
  `SELECT ... FOR UPDATE` row locks scattered across usecases.

Don't raise the isolation level of a transaction "to be safe" — it costs
throughput and, at `SERIALIZABLE`, requires the caller to handle retryable
conflict errors. Default to Read Committed + optimistic locking (the
`version` column pattern) unless a query genuinely needs a consistent
multi-statement snapshot.
