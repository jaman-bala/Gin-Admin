# Backend — Go/Gin REST API

Clean-architecture REST API for the Gin Admin panel. Handles authentication, user management, RBAC, audit logging, file uploads, and analytics.

## Tech stack

- **Go 1.25** + **Gin** — HTTP framework
- **PostgreSQL 17** — primary database (sqlx)
- **Redis** — token blacklist + rate limiting
- **MinIO** — S3-compatible object storage for avatars
- **Goose** — SQL migrations
- **Swaggo** — OpenAPI spec generation (served via Scalar UI)

## Configuration

Copy the example and fill in your values:

```bash
cp .env.example .env
```

Key variables:

| Variable | Purpose |
|---|---|
| `SERVER_PORT` | HTTP listen port |
| `DB_*` | PostgreSQL connection |
| `REDIS_HOST`, `REDIS_PORT` | Redis connection |
| `MINIO_HOST` | MinIO host **inside Docker network** |
| `MINIO_PUBLIC_HOST` | MinIO host **visible to the browser** (for presigned URLs) |
| `SECRET_KEY` | JWT signing secret (min 32 chars) |
| `ADMIN_DEFAULT_PHONE` | Phone number for the auto-created superuser |
| `ADMIN_DEFAULT_PASSWORD` | Password for the auto-created superuser |

> `MINIO_HOST` and `MINIO_PUBLIC_HOST` serve different purposes: the backend uses `MINIO_HOST` to connect to MinIO internally (e.g., `minio` in Docker), while presigned URLs returned to clients use `MINIO_PUBLIC_HOST` (e.g., `localhost:9000`).

## Running locally

```bash
# Start dependencies (postgres, redis, minio) via docker compose or locally
# then:
make run
```

API will be available at `http://localhost:<SERVER_PORT>`.
API docs (Scalar): `http://localhost:<SERVER_PORT>/docs`

## Commands

```bash
make build           # compile bin/api and bin/migrate
make run             # go run cmd/api/main.go
make test            # go test -v ./...
make swag            # regenerate docs/ from source annotations

make migrate-up      # apply all pending migrations
make migrate-down    # rollback last migration
make migrate-status  # show migration state
make migrate-create name=create_something_table
```

## Architecture

```
cmd/
  api/main.go        entrypoint — loads config, wires deps, starts HTTP
  migrate/main.go    goose migration runner
config/config.go     env-based config (reads Docker secrets first)
internal/
  domain/            entities + repository interfaces (no external deps)
  application/       business logic use-cases (depend only on domain)
  infrastructure/
    postgres/        sqlx repository implementations
    redis/           token blacklist + rate limit
    minio/           object storage
    http/
      router.go      Gin setup, middleware wiring, route registration
      handler/       thin HTTP handlers
      middleware/    auth, blacklist, audit, roles, rate-limit, sanitize
pkg/errors/          sentinel error values
migrations/          goose SQL files
docs/                generated Swagger/OpenAPI spec
```

All dependency wiring happens in `router.go:buildDeps()`. Handlers never call infrastructure directly.

## API overview

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/login` | — | Login, returns token pair |
| POST | `/api/v1/auth/logout` | Bearer | Blacklist tokens |
| POST | `/api/v1/auth/refresh` | — | Refresh access token |
| GET | `/api/v1/users/me` | Bearer | Own profile |
| PUT | `/api/v1/users/me` | Bearer | Update own profile + avatar |
| GET | `/api/v1/users` | Admin | List users (paginated, search) |
| POST | `/api/v1/users` | Admin | Create user |
| PATCH | `/api/v1/users/:id` | Admin | Update user |
| DELETE | `/api/v1/users/:id` | Admin | Soft-delete user |
| GET | `/api/v1/users/stats` | Admin | Analytics (total, active, admins, new) |
| GET | `/api/v1/audit` | Admin | Audit log (paginated) |
| GET | `/health` | — | Health check (db, redis, minio) |

Full interactive docs: `http://localhost:<SERVER_PORT>/docs`

## Middleware chain

All requests: `RequestID → Metrics → CORS → Sanitize`

Auth routes: `+ AuditMid` (login also adds `LoginRateLimit`)

User routes: `+ TokenBlacklist → AuthMid → AuditMid`

Admin routes: `+ TokenBlacklist → AuthMid → RequireRoleLevel(admin) → AuditMid`

## Role levels

`user (1) < admin (2) < superuser (3)` — enforced by `RequireRoleLevelMiddleware`.

Token auth accepts both `Authorization: Bearer <token>` header and `access_token` cookie.

## Database schema

- `users` — UUID PK, phone (unique), bcrypt password, role, photo URL, soft-delete via `deleted_at`
- `audit_logs` — serial PK, user_id FK, action/entity/entity_id, client_ip, user_agent, status