# Workspace

## Overview

pnpm workspace monorepo using TypeScript. Full-stack Auth Dashboard with JWT authentication and user management.

## Stack

- **Monorepo tool**: pnpm workspaces
- **Node.js version**: 24
- **Package manager**: pnpm
- **TypeScript version**: 5.9
- **API framework**: Express 5
- **Database**: PostgreSQL + Drizzle ORM
- **Validation**: Zod (`zod/v4`), `drizzle-zod`
- **API codegen**: Orval (from OpenAPI spec)
- **Build**: esbuild (CJS bundle)
- **Frontend**: React + Vite + TailwindCSS + shadcn/ui
- **State management**: Zustand (auth state with localStorage persistence)
- **Auth**: JWT (access token 15m + refresh token 30d), bcryptjs for passwords

## Key Commands

- `pnpm run typecheck` — full typecheck across all packages
- `pnpm run build` — typecheck + build all packages
- `pnpm --filter @workspace/api-spec run codegen` — regenerate API hooks and Zod schemas from OpenAPI spec
- `pnpm --filter @workspace/db run push` — push DB schema changes (dev only)
- `pnpm --filter @workspace/api-server run dev` — run API server locally

## Artifacts

- **auth-dashboard** (`artifacts/auth-dashboard/`) — React + Vite frontend at `/`
- **api-server** (`artifacts/api-server/`) — Express 5 backend at `/api`

## Auth System

### Backend (`artifacts/api-server/`)
- `src/lib/auth.ts` — JWT signing/verification, bcrypt hashing, token helpers
- `src/middlewares/authenticate.ts` — Bearer token middleware, admin role check
- `src/routes/auth.ts` — POST /auth/login, /auth/register, /auth/refresh, /auth/logout
- `src/routes/users.ts` — CRUD users, /users/me, /users/stats (admin only)

### Database Schema (`lib/db/src/schema/`)
- `users.ts` — Users table with role (admin/user), phone, passwordHash, photo, isActive
- `refreshTokens.ts` — Refresh token store with revocation support

### Frontend (`artifacts/auth-dashboard/src/`)
- `store/authStore.ts` — Zustand store: user, accessToken, refreshToken, isAuthenticated
- `lib/axiosClient.ts` — Axios instance with auto-refresh interceptor on 401
- `lib/apiClient.ts` — setAuthTokenGetter for Orval-generated hooks
- Pages: /login, /register, /dashboard, /users (admin), /profile

## Seed Accounts

- Admin: phone `+1000000001`, password `admin123`
- User: phone `+1000000002`, password `user123`

See the `pnpm-workspace` skill for workspace structure, TypeScript setup, and package details.
