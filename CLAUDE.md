# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A fullstack admin panel template with a Go/Gin REST API backend and a React/TypeScript frontend. The backend handles JWT authentication, RBAC, audit logging, file storage, and analytics. The frontend is an auth dashboard built with Vite + React + Tailwind CSS + shadcn/ui.

## Commands

### Backend (`backend/`)

```bash
# Run development server
make run                    # go run cmd/api/main.go

# Build binaries
make build                  # produces bin/api and bin/migrate

# Tests
make test                   # go test -v ./...
go test -v ./internal/...   # run a single package: go test -v ./internal/application/auth/...

# Linting
go vet ./...
golangci-lint run ./...

# Swagger docs (requires swag CLI)
make swag

# Database migrations
make migrate-up
make migrate-down
make migrate-status
make migrate-create name=create_something_table
```

### Frontend (`frontend/`)

Uses **pnpm** exclusively (enforced via preinstall script).

```bash
# Install dependencies (from frontend/)
pnpm install

# Dev server (запускается из frontend/)
pnpm dev          # http://localhost:5173

# Build all artifacts
pnpm build

# Typecheck
pnpm typecheck
```

### Docker (full stack)

```bash
# Copy and fill .env
cp backend/.env.example .env

# Start all services (backend, frontend, postgres, redis, minio)
docker compose up --build
```

Services: backend on `$SERVER_PORT`, frontend on `8085`, postgres on `5412`, redis on `6312`, minio on `9000/9001`.

## Architecture

### Backend — Clean Architecture layers

```
cmd/
  api/main.go          — entrypoint: loads config, init DB, start HTTP server
  migrate/main.go      — goose migration runner
  worker/main.go       — background worker (optional)
config/config.go       — env-based config (Server, Database, JWT, Redis, Minio)
internal/
  domain/              — pure entities + repository interfaces (no deps)
    user/              — User entity (roles: superuser > admin > user), Repository
    auditlog/          — AuditLog entity + Repository
    analytics/         — Analytics entity + Repository
    token/             — Token repository (Redis-backed)
    file/              — File repository (MinIO-backed)
  application/         — business logic usecases (depend only on domain interfaces)
    auth/              — Login, Logout, RefreshToken, GetUserInfoFromToken
    user/              — CRUD, GetMe, UpdateMe
    auditlog/          — logging writes/reads
    analytics/         — user stats
    token/             — GenerateTokenPair, BlacklistToken, GetTokenInfo
    file/              — upload/delete via MinIO
  infrastructure/
    postgres/          — sqlx implementations of domain repositories
    redis/             — token blacklist + rate limit cache
    minio/             — object storage implementation
    http/
      router.go        — Gin engine setup, CORS, middleware wiring, route registration
      handler/         — thin HTTP handlers (auth, user, audit, analytics, health)
      middleware/       — auth, blacklist, audit, roles, rate-limit, sanitize, metrics, requestid
pkg/
  errors/errors.go     — sentinel error values
migrations/            — goose SQL migrations
docs/                  — swaggo-generated Swagger docs
```

**Dependency wiring** happens entirely in `router.go:buildDeps()`. All application services are constructed there and injected into handlers. Handlers never call infrastructure directly.

**Secrets** are read from `/run/secrets/<lowercase_key>` first (Docker Swarm secrets), then from environment variables — see `config.readSecret()`.

### Middleware chain (per request)

`RequestID → Metrics → CORS → Sanitize` for all routes, then per-group:
- Auth routes: `AuditMid` + optionally `LoginRateLimit`, `TokenBlacklist`, `AuthMid`
- User routes: `TokenBlacklist → AuthMid → AuditMid`, with admin subroutes adding `RequireRoleLevel(admin)`
- Audit routes: `TokenBlacklist → AuthMid → RequireRoleLevel(admin) → AuditMid`

**Token auth** accepts both `Authorization: Bearer <token>` header and `access_token` cookie.

**Role levels**: `user(1) < admin(2) < superuser(3)` — enforced by `RequireRoleLevelMiddleware`.

### Frontend — pnpm workspace

```
frontend/
  artifacts/
    auth-dashboard/    — main React app (Vite, Tailwind, shadcn/ui, Zustand, React Query)
    api-server/        — lightweight API server artifact
    mockup-sandbox/    — UI component sandbox
  lib/
    api-spec/          — OpenAPI spec + orval config (generates typed API client)
    api-client-react/  — generated React Query hooks from OpenAPI spec
    api-zod/           — Zod schemas generated from OpenAPI spec
    db/                — Drizzle config (frontend-side DB if needed)
```

The frontend resolves `@workspace/api-client-react` via the pnpm workspace, so the generated client is shared across artifacts without publishing.

**Routing** in auth-dashboard uses `wouter`. State management uses `zustand`. API calls use `axios` + `@tanstack/react-query`. UI components are shadcn/ui (Radix UI primitives + Tailwind).

### Database schema

- `users` — UUID PK, phone (unique), bcrypt password, role, photo URL, soft-delete via `deleted_at`
- `audit_logs` — serial PK, references user_id, action/entity/entity_id, client_ip, user_agent, status

Migrations managed with **goose** (SQL files in `backend/migrations/`).