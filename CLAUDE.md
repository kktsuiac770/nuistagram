# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

NUIstagram is a photo-sharing app with a Go backend and React/TypeScript frontend.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, net/http, SQLite (go-sqlite3) |
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
├── Makefile                         # make dev / make build / make backend / make frontend
├── backend/                         # Go module root (go.mod here)
│   ├── cmd/server/main.go           # Binary entry point
│   ├── internal/
│   │   ├── cache/                   # In-memory TTL cache
│   │   ├── config/                  # YAML config + env var overrides
│   │   ├── database/                # SQLite init, schema, auto-migrations
│   │   ├── middleware/              # Recovery, RequestID, logging, tracing
│   │   ├── models/                  # Domain structs (User, Photo, Comment, …)
│   │   ├── monitoring/              # Prometheus-compatible metrics
│   │   ├── repository/              # Interface definitions + SQLite implementations
│   │   │   └── mocks/               # Mock implementations for handler tests
│   │   ├── server/                  # HTTP handlers + *_test.go unit tests
│   │   ├── session/                 # Session manager (24h TTL, 1h inactivity)
│   │   ├── storage/                 # Pluggable image storage (local / Cloudinary)
│   │   └── tests/                   # E2E integration tests (real in-memory SQLite)
│   ├── config.example.yaml          # Config template
│   └── static/uploads/              # User-uploaded images (git-ignored; must exist)
├── frontend/
│   └── src/
│       ├── pages/                   # Route-level components
│       ├── components/              # Shared UI components
│       ├── contexts/                # AuthContext, ThemeContext
│       ├── hooks/                   # useFollowStatus, …
│       ├── lib/api.ts               # Centralised API client + CSRF handling
│       └── test/setup.ts            # Vitest global setup
├── data/                            # SQLite DB file (git-ignored; must exist)
└── static/                          # Static file serving root
```

---

## Build & Run Commands

### Makefile (from repo root)

```bash
make dev        # Start backend + frontend concurrently (creates data/ and static/uploads/ if missing)
make backend    # Backend only (go run ./cmd/server from backend/)
make frontend   # Frontend only (npm run dev from frontend/)
make build      # Build backend binary + frontend bundle
```

### Backend (run from `backend/`)

```bash
go build -o nuistagram ./cmd/server   # Compile binary
go run ./cmd/server                   # Run without building
go fmt ./... && go vet ./...          # Format + static analysis
go test ./...                         # All tests
go test ./internal/server -v          # Handler unit tests (verbose)
go test -run TestRegister_Success ./internal/server   # Single test
go test -run "TestLogin.*" ./internal/server          # Pattern match
go test -v ./... -race -coverprofile=coverage.out     # Race + coverage
```

### Frontend (run from `frontend/`)

```bash
npm install          # Install dependencies
npm run dev          # Dev server on :5173 (proxies API to :8080)
npm run build        # tsc + vite → frontend/dist/
npm run lint         # ESLint check
npm run test:run     # Single pass
npm run test:run -- src/contexts/Auth.test.tsx  # Single file
npm run test:coverage  # Coverage report
```

> **Before every commit:** `go fmt ./... && npm run lint` (CI enforces both).

> **First-time setup:** `mkdir -p backend/data backend/static/uploads` (or just run `make dev`).

---

## Configuration

Config is loaded from `backend/config.yaml` (create from `config.example.yaml`). Environment variables override YAML values:

| Env Var | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `data/nuistagram.db` | SQLite database path |
| `SECURE_COOKIE` | `false` | Set `true` in production (HTTPS) |
| `ENV` | `development` | Affects log format (`production` → JSON) |
| `STORAGE_PROVIDER` | `local` | `local` or `cloudinary` |
| `STORAGE_UPLOAD_DIR` | `static/uploads` | Local upload directory |
| `CLOUDINARY_CLOUD_NAME` | — | Cloudinary cloud name |
| `CLOUDINARY_API_KEY` | — | Cloudinary API key |
| `CLOUDINARY_API_SECRET` | — | Cloudinary API secret |
| `CLOUDINARY_FOLDER` | `nuistagram` | Cloudinary path prefix |

---

## Image Storage

Storage is pluggable via `backend/internal/storage/`. The `Storage` interface (`Upload`, `Delete`, `FetchContent`) is selected at startup by `STORAGE_PROVIDER`:

- **`local`**: writes to `static/uploads/`; returns bare filename as identifier.
- **`cloudinary`**: uploads via signed SHA-1 API; returns full HTTPS URL as identifier.

Image processing happens automatically on upload (via `github.com/disintegration/imaging` and `github.com/dsoprea/go-exif`):
- Photos: compressed to max 1920px wide, 85% JPEG quality; thumbnails at 400×400px
- Avatars: cropped/resized to 200×200px square
- EXIF date extracted and stored with photo metadata

---

## Database

- **Engine:** SQLite, initialised automatically on first start.
- **Location:** `data/nuistagram.db` (relative to where the binary runs; git-ignored).
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

    "nuistagram/internal/models"
)
```

### Naming

- Exported symbols: `PascalCase`; unexported: `camelCase`
- Repository interfaces: `UserRepository`; implementations: `userRepository`
- Constructors: `NewUserRepository(db *sql.DB) UserRepository`
- Handlers live in `internal/server/` (e.g. `Login`, `Register`, `APIGetPhotos`)

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

Use `type` keyword for type-only imports. Path alias `@/` maps to `frontend/src/`.

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

Use the `api` object from `lib/api.ts` — never call `fetch` directly in components. CSRF tokens are handled automatically by `fetchWithCsrf()` inside `api.ts`.

---

## Authentication & Security

- **Mechanism:** Session cookie (HTTP-only, SameSite=Strict, 24h expiry, 1h inactivity timeout).
- **In handlers:** Call `GetCurrentUser(r)` to get the authenticated user; returns `nil` for unauthenticated requests.
- **CSRF:** All state-changing POST requests require a CSRF token (session-scoped, not per-request). Use `GET /api/csrf-token` to fetch. Frontend `fetchWithCsrf()` manages this automatically.
- **Password rules:** ≥ 8 chars, at least one uppercase, one lowercase, one digit.
- **Rate limiting:** Login and register endpoints are rate-limited.

---

## API Conventions

### Endpoint patterns

| Category | Examples |
|---|---|
| REST API (JSON) | `GET /api/photos`, `GET /api/photo/{id}`, `GET /api/me` |
| Auth (form) | `POST /login`, `POST /register`, `GET /logout` |
| Mutations (form/multipart) | `POST /upload`, `POST /photo/{id}/delete`, `POST /photo/{id}/edit` |
| Health | `GET /healthz`, `GET /readyz` |
| Export | `GET /export` — ZIP of all user's photos |

### Key API endpoints

```
GET  /api/csrf-token                  # Get CSRF token for current session
GET  /api/photos                      # Feed (paginated; ?page=1&tags=x&feed=following)
GET  /api/photo/{id}                  # Single photo
POST /api/photo/{id}/like             # Toggle like (CSRF required)
GET  /api/photo/{id}/comments         # Comments
POST /api/photo/{id}/comment          # Add comment
GET  /api/photo/{id}/likers           # Who liked this photo
GET  /api/me                          # Current user with counts
GET  /api/user/{username}             # User profile with follow status
GET  /api/user/{username}/photos      # User's photos
GET  /api/user/{username}/follow-status
POST /api/user/{username}/follow      # Follow (CSRF required)
POST /api/user/{username}/unfollow    # Unfollow (CSRF required)
GET  /api/notifications               # Notifications list
GET  /api/notifications/unread        # Unread count
POST /api/notifications/{id}/read     # Mark one read (CSRF required)
POST /api/notifications/read-all      # Mark all read (CSRF required)
GET  /api/search/users                # User search
POST /api/profile                     # Update bio (CSRF required)
POST /api/avatar                      # Upload avatar (CSRF required)
GET  /api/nuis                        # List all tags
GET  /export                          # Download ZIP of all user photos
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

- Place unit tests in `backend/internal/server/*_test.go`.
- Use `mocks.NewMockRepositories()` — never a real database in unit tests.
- E2E tests in `backend/internal/tests/` use a real in-memory SQLite database.

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
4. **Tracing** — span tracking for observability

---

## CI/CD

Two GitHub Actions workflows:

| Workflow | Trigger | Jobs |
|---|---|---|
| `ci.yml` | push to `main`, PRs | go fmt check, go vet, go build, go test; npm ci, lint, build |
| `test.yml` | push / PRs | go test with race + coverage; frontend test:run + build; combined binary artifact upload |

CI fails if `go fmt` produces a diff, `go vet` reports issues, any test fails, ESLint errors, or frontend build fails.

---

## Important Notes

- **Serving frontend:** The binary serves `frontend/dist/` as a SPA if it exists; otherwise falls back to `templates/`.
- **"Nuis":** The app's term for photo tags/categories.
- **No ORM:** All SQL is hand-written inside `internal/repository/`. Keep queries there.
- **No global state in frontend:** Use `AuthContext` for user session and React Query for server data.
- **Storage identifiers differ by provider:** local returns a filename; Cloudinary returns a full URL. Handlers use `helpers.go` utilities (`imageURL`) to construct the right URL for responses.
