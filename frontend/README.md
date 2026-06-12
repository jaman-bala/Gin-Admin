# Frontend — React/TypeScript Admin Dashboard

pnpm workspace containing the admin dashboard UI and supporting packages. Built with Vite, Tailwind CSS, shadcn/ui, and a typed API client generated from the backend's OpenAPI spec.

## Tech stack

- **React 19** + **TypeScript**
- **Vite** — build tool and dev server
- **Tailwind CSS** + **shadcn/ui** (Radix UI primitives)
- **Zustand** — auth state management
- **TanStack React Query** + **axios** — data fetching
- **Recharts** — charts (dashboard activity + role distribution)
- **wouter** — lightweight client-side routing
- **orval** — generates typed React Query hooks from OpenAPI spec
- **pnpm workspaces** — shared packages without publishing

## Prerequisites

- Node.js 20+
- pnpm 10+ (`npm install -g pnpm`)

## Getting started

```bash
# from the frontend/ directory
pnpm install
pnpm dev          # starts auth-dashboard at http://localhost:5173
```

The dev server proxies `/api` requests to `http://localhost:8082` (configured in `vite.config.ts`). Start the backend first or adjust the proxy target.

## Workspace layout

```
frontend/
  artifacts/
    auth-dashboard/    Main admin panel app (this is what you deploy)
    api-server/        Lightweight API proxy artifact
    mockup-sandbox/    UI component sandbox for isolated development
  lib/
    api-spec/          OpenAPI spec + orval config
    api-client-react/  Generated React Query hooks (do not edit manually)
    api-zod/           Zod schemas generated from OpenAPI spec
    db/                Drizzle config (frontend DB if needed)
  package.json         Workspace root — pnpm filter scripts
```

## Available scripts

From `frontend/` (workspace root):

```bash
pnpm dev            # run auth-dashboard dev server
pnpm build          # build auth-dashboard for production
pnpm typecheck      # typecheck auth-dashboard
```

From `artifacts/auth-dashboard/`:

```bash
pnpm dev
pnpm build
pnpm typecheck
```

## Regenerating the API client

After changing the backend's OpenAPI spec:

```bash
cd lib/api-spec
pnpm generate       # runs orval → updates api-client-react/
```

The generated hooks in `lib/api-client-react/` are committed to the repo so the dashboard can import them without a build step.

## Pages

| Route | Access | Description |
|---|---|---|
| `/login` | Public | Phone + password login |
| `/register` | Public | New account registration |
| `/dashboard` | Auth | Stats, activity chart, quick actions |
| `/users` | Admin | User table, create/edit/delete |
| `/audit` | Admin | Paginated audit log |
| `/profile` | Auth | Own profile + avatar upload |

## Auth flow

Tokens are stored in Zustand (`authStore`) and persisted to `localStorage`. The axios instance automatically attaches the access token and handles 401 responses by attempting a refresh, then redirecting to `/login` on failure.

## Adding new pages

1. Create `src/pages/my-page.tsx`
2. Add a route in `src/App.tsx`
3. Add a nav item in `src/components/layout/AppLayout.tsx` (the `navItems` array)

## Docker

The Dockerfile builds the static dist and serves it via nginx:

```bash
# from project root
docker compose up --build frontend
```

Output is served at `http://localhost:8085`.