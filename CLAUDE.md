# CLAUDE.md — NUIstagram

AI assistant guidelines for NUIstagram, a photo-sharing app with a Go backend and React/TypeScript frontend.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, net/http, SQLite (go-sqlite3) |
| Frontend | React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4 |
| Testing (BE) | testify (mocks + assertions) |
| Testing (FE) | Vitest 4, React Testing Library 16 |
| State | React Query (server state), Context API (auth, theme) |
| Auth | Session cookies (HTTP-only), CSRF tokens |
| Observability | slog (structured logging), Prometheus metrics, custom tracing |

---

## Repository Layout

```
nuistagram/
├── cmd/server/main.go           # Binary entry point
├── internal/
│   ├── cache/                   # In-memory TTL cache
│   ├── config/                  # YAML config + env var overrides
│   ├── database/                # SQLite init + schema creation
│   ├── handlers/                # HTTP handlers + *_test.go unit tests
│   ├── logging/                 # Structured logging helpers
│   ├── metrics/                 # Prometheus-compatible metrics
│   ├── middleware/              # Recovery, RequestID, logging, tracing
│   ├── models/                  # Domain structs (User, Photo, Comment, …)
│   ├── repository/              # Interface definitions + SQLite implementations
│   │   └── mocks/               # Mock implementations for handler tests
│   └── tests/                   # E2E integration tests (real SQLite)
├── frontend/
│   └── src/
│       ├── pages/               # Route-level components
│       ├── components/          # Shared UI components (Toast, …)
│       ├── contexts/            # AuthContext, ThemeContext
│       ├── hooks/               # useFollowStatus, …
│       ├── lib/api.ts           # Centralised API client + CSRF handling
│       └── test/setup.ts        # Vitest global setup
├── templates/                   # Server-side HTML fallback templates
├── static/uploads/              # User-uploaded images (git-ignored)
├── data/                        # SQLite DB file (git-ignored)
├── config.example.yaml          # Config template
├── .github/workflows/           # CI: ci.yml (lint+build), test.yml (tests)
├── AGENTS.md                    # Concise quick-reference for agents
└── README.md                    # Project overview
```

---

## Build & Run Commands

### Backend (run from repo root)

```bash
go build -o nuistagram ./cmd/server   # Compile binary
./nuistagram                          # Run on :8080
go fmt ./... && go vet ./...          # Format + static analysis
go test ./...                         # All tests
go test ./internal/handlers -v        # Handler unit tests (verbose)
go test -run TestRegister_Success ./internal/handlers   # Single test
go test -run "TestLogin.*" ./internal/handlers          # Pattern match
go test -v ./... -race -coverprofile=coverage.out       # Race + coverage
```

### Frontend (run from `frontend/`)

```bash
npm install          # Install dependencies
npm run dev          # Dev server on :5173 (proxies API to :8080)
npm run build        # tsc + vite → frontend/dist/
npm run lint         # ESLint check
npm run test         # Vitest watch mode
npm run test:run     # Single pass
npm run test:run -- src/contexts/Auth.test.tsx  # Single file
npm run test:coverage  # Coverage report
npm run preview      # Serve production build locally
```

> **Before every commit:** `go fmt ./... && npm run lint` (CI enforces both).

---

## Configuration

Config is loaded from `config.yaml` (create from `config.example.yaml`).
Environment variables override YAML values:

| Env Var | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `data/nuistagram.db` | SQLite database path |
| `SECURE_COOKIE` | `false` | Set `true` in production (HTTPS) |
| `ENV` | `development` | Affects log format (`production` → JSON) |

---

## Database

- **Engine:** SQLite, initialised automatically on first start.
- **Location:** `data/nuistagram.db` (git-ignored; created at runtime).
- **Schema tables:** `users`, `photos`, `nuis`, `photo_nuis`, `favorites`, `likes`, `comments`, `follows`, `notifications`, `albums`, `album_photos`.
- **Migrations:** `migrateToMultiNui()` and `migrateUserProfile()` run automatically in `internal/database/`.
- Never hand-write SQL outside of `internal/database/` or `internal/repository/`.

---

## Go Code Conventions

### Import grouping (always three groups, blank lines between)

```go
import (
    "database/sql"
    "net/http"

    "github.com/stretchr/testify/assert"
    "golang.org/x/crypto/bcrypt"

    "nuistagram/internal/models"
)
```

### Naming

- Exported symbols: `PascalCase`; unexported: `camelCase`
- Repository interfaces: `UserRepository`; implementations: `userRepository`
- Constructors: `NewUserRepository(db *sql.DB) UserRepository`
- Handlers: `Login`, `Register`, `APIGetPhotos`

### Error handling

```go
// Repository layer — return errors
func (r *userRepository) GetByID(id int64) (*models.User, error) { ... }

// Check for not-found
if err == sql.ErrNoRows { ... }

// Handler JSON errors — always set Content-Type first
w.Header().Set("Content-Type", "application/json")
http.Error(w, `{"error": "not found"}`, http.StatusNotFound)
```

### Repository pattern

All database access goes through repository interfaces. Never call `*sql.DB` directly from handlers.

```go
// In handlers, access data through Repos global
user, err := Repos.Users.GetByID(userID)

// Initialise in main
repos := repository.NewRepositories(db, cache)
```

---

## TypeScript / React Conventions

### Imports

```typescript
import { useState, useEffect, type ReactNode } from 'react'
import { api, type User } from '../lib/api'
```

Use `type` keyword for type-only imports.
Path alias `@/` maps to `frontend/src/`.

### Naming

- Components/pages: `PascalCase` filenames (`PhotoDetail.tsx`)
- Hooks: `useAuth`, `useFollowStatus`
- Contexts: `AuthContext`, `AuthProvider`
- Interfaces/types: `Photo`, `User`, `PhotosResponse`

### Component structure (order matters)

1. Hook calls
2. React Query queries / mutations
3. Early returns for loading / error states
4. Main JSX return

### Styling

Tailwind CSS 4 with dark mode support. Always include dark variants:

```tsx
<div className="bg-white dark:bg-black text-gray-900 dark:text-white">
```

### API client

Use the `api` object from `lib/api.ts` — never call `fetch` directly in components.
CSRF tokens are handled automatically by `fetchWithCsrf()` inside `api.ts`.

```typescript
const photos = await api.getPhotos({ page: 1 })
await api.toggleLike(photoId)
```

---

## Authentication & Security

- **Mechanism:** Session cookie (HTTP-only, SameSite=Strict, 24 h expiry).
- **In handlers:** Call `GetCurrentUser(r)` to get the authenticated user; returns `nil` for unauthenticated requests.
- **CSRF:** All state-changing POST requests require a CSRF token. Frontend `fetchWithCsrf()` manages this automatically.
- **Password rules:** ≥ 8 chars, at least one uppercase, one lowercase, one digit (enforced at registration).
- **Secure flag:** Enabled via `SECURE_COOKIE=true` in production.

---

## API Conventions

### Endpoint patterns

| Category | Examples |
|---|---|
| REST API (JSON) | `GET /api/photos`, `GET /api/photo/{id}`, `GET /api/me`, `GET /api/user/{username}` |
| Auth (form) | `POST /login`, `POST /register`, `/logout` |
| Mutations (form) | `POST /upload`, `POST /photo/{id}/delete`, `POST /photo/{id}/edit` |
| Health | `/healthz`, `/readyz`, `/metrics` |

### Request / response

- Auth and upload forms: `application/x-www-form-urlencoded` or `multipart/form-data`
- All `/api/` responses: `application/json` with `snake_case` keys
- Pagination: `?page=1`
- Filtering: `?tags=nature,travel&feed=following`

### Key API endpoints

```
GET  /api/photos                      # Feed (paginated)
GET  /api/photo/{id}                  # Single photo
POST /api/photo/{id}/like             # Toggle like
GET  /api/photo/{id}/comments         # Comments
POST /api/photo/{id}/comment          # Add comment
GET  /api/user/{username}             # User profile
POST /api/user/{username}/follow      # Follow
POST /api/user/{username}/unfollow    # Unfollow
GET  /api/notifications               # Notifications list
GET  /api/notifications/unread        # Unread count
POST /api/notifications/read-all      # Mark all read
GET  /api/search/users                # User search
POST /api/profile                     # Update bio
POST /api/avatar                      # Upload avatar
GET  /api/nuis                        # List all tags
```

---

## Testing Patterns

### Backend — handler unit tests

```go
func setupAuthMocks() *mocks.MockRepositories {
    mockRepos := mocks.NewMockRepositories()
    Repos = &repository.Repositories{
        Users:         mockRepos.Users,
        Photos:        mockRepos.Photos,
        Notifications: mockRepos.Notifications,
    }
    return mockRepos
}

func TestRegister_Success(t *testing.T) {
    mockRepos := setupAuthMocks()
    mockRepos.Users.On("Create", "testuser", mock.AnythingOfType("string")).
        Return(int64(1), nil)

    // ... exercise handler via httptest ...

    mockRepos.Users.AssertExpectations(t)
}
```

- Place unit tests in `internal/handlers/*_test.go`.
- Use `mocks.NewMockRepositories()` — never a real database in unit tests.
- E2E tests in `internal/tests/` use a real in-memory SQLite database.

### Frontend — component / hook tests

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import * as apiModule from '../lib/api'

beforeEach(() => vi.clearAllMocks())

it('shows username after login', async () => {
    vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'alice' })

    render(<AuthProvider><TestComponent /></AuthProvider>)

    await waitFor(() => expect(screen.getByText('alice')).toBeInTheDocument())
})
```

- Use `vi.spyOn` on `apiModule.api` — never mock `fetch` globally.
- Always `await waitFor(...)` for async state changes.
- `beforeEach(() => vi.clearAllMocks())` is mandatory to prevent test pollution.

---

## Middleware Stack (request order)

1. **Recovery** — catches panics, returns 500
2. **RequestID** — attaches `X-Request-ID` correlation header
3. **Logging** — structured request/response logging via slog
4. **Metrics** — Prometheus counters/histograms
5. **Tracing** — span tracking for observability

---

## CI/CD

Two GitHub Actions workflows:

| Workflow | Trigger | Jobs |
|---|---|---|
| `ci.yml` | push to `main`, PRs | go fmt check, go vet, go build, go test; npm ci, lint, build |
| `test.yml` | push / PRs | go test with race + coverage; frontend test:run + build; combined binary artifact upload |

CI will fail if:
- `go fmt` produces any diff
- `go vet` reports issues
- Any Go test fails
- ESLint reports errors
- Frontend build fails

---

## Important Notes

- **Serving frontend:** The binary serves `frontend/dist/` as a SPA if it exists; otherwise falls back to `templates/`.
- **Uploads:** Stored in `static/uploads/` (git-ignored). The directory must exist before the server starts.
- **Database dir:** `data/` must exist; create it with `mkdir -p data` if missing.
- **Nuis:** The app's term for photo tags/categories (not standard hashtags).
- **No ORM:** All SQL is hand-written inside `internal/repository/`. Keep queries there.
- **No global state in frontend:** Use `AuthContext` for user session and React Query for server data.
