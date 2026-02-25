# AGENTS.md

Coding agent guidelines for NUIstagram - a photo sharing app with Go backend and React frontend.

## Build Commands

### Backend (Go)
```bash
go build -o nuistagram ./cmd/server              # Build
./nuistagram                                      # Run (port 8080)
go test ./...                                     # All tests
go test ./internal/handlers -v                    # Package tests
go test -run TestRegister_Success ./internal/handlers  # Single test
go test -run "TestLogin.*" ./internal/handlers    # Pattern match
go fmt ./... && go vet ./...                      # Format + lint
```

### Frontend (React + TypeScript)
```bash
cd frontend
npm install                                        # Install deps
npm run dev                                        # Dev server (port 5173)
npm run build                                      # Production build
npm run lint                                       # ESLint
npm run test                                       # Vitest watch mode
npm run test:run                                   # Run once
npm run test:run -- src/contexts/Auth.test.tsx     # Single test file
npm run test:coverage                              # With coverage
```

## Project Structure

```
nuistagram/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── cache/                   # Caching layer
│   ├── database/                # DB initialization
│   ├── handlers/                # HTTP handlers + *_test.go
│   ├── models/                  # Data models
│   ├── repository/              # Interfaces + implementations
│   │   └── mocks/               # Mock implementations
│   └── tests/                   # E2E integration tests
├── frontend/src/
│   ├── pages/                   # Route pages
│   ├── components/              # Reusable components
│   ├── contexts/                # React contexts (Auth, Theme)
│   ├── lib/api.ts               # API client
│   └── test/setup.ts            # Test setup
├── static/uploads/              # User uploads
├── templates/                   # HTML fallback
└── data/                        # SQLite database
```

## Go Code Style

**Imports:** Group with blank lines: stdlib → external → internal
```go
import (
    "database/sql"
    "net/http"
    
    "github.com/stretchr/testify/assert"
    "golang.org/x/crypto/bcrypt"
    
    "nuistagram/internal/models"
)
```

**Naming:**
- Exported: `PascalCase`, unexported: `camelCase`
- Interfaces: `UserRepository`, implementations: `userRepository`
- Constructors: `NewUserRepository(db *sql.DB) UserRepository`
- Handlers: `Login`, `Register`, `APIGetPhotos`

**Error Handling:**
- Return errors from repository functions
- Check `sql.ErrNoRows` for not-found
- JSON errors: `http.Error(w, \`{"error": "msg"}\`, http.StatusBadRequest)`
- Always set `Content-Type: application/json` for API responses

**Repository Pattern:**
- All DB access through repository interfaces
- Use `Repos.Users.GetByID(id)` in handlers
- Initialize via `repository.NewRepositories(db, cache)`

## TypeScript/React Code Style

**Imports:** Type-only imports for types
```typescript
import { useState, useEffect, type ReactNode } from 'react'
import { api, type User } from '../lib/api'
```

**Naming:**
- Components: `PhotoDetail.tsx` (PascalCase)
- Hooks: `useAuth`, `useTheme`
- Contexts: `AuthContext`, `AuthProvider`
- Interfaces: `Photo`, `User`, `PhotosResponse`

**Component Structure:**
1. Hooks at top
2. Queries/mutations
3. Early returns for loading/error
4. Main render at bottom

**Styling:** Tailwind CSS with dark mode support
```typescript
<div className="bg-white dark:bg-black text-gray-900 dark:text-white">
```

**API Client:** Use `api` object from `lib/api.ts`
- `api.getPhotos()`, `api.login()`, `api.toggleLike(id)`
- CSRF token handled automatically via `fetchWithCsrf()`

## Testing

### Backend Tests

**Unit Tests** (`internal/handlers/*_test.go`):
```go
func setupAuthMocks() *mocks.MockRepositories {
    mockRepos := mocks.NewMockRepositories()
    Repos = &repository.Repositories{
        Users: mockRepos.Users,
        // ... other repos
    }
    return mockRepos
}

func TestRegister_Success(t *testing.T) {
    mockRepos := setupAuthMocks()
    mockRepos.Users.On("Create", "testuser", mock.AnythingOfType("string")).Return(int64(1), nil)
    // ... test code
    mockRepos.Users.AssertExpectations(t)
}
```

**E2E Tests** (`internal/tests/`): Use real SQLite database

### Frontend Tests

**Setup:** Vitest + React Testing Library, jsdom environment

**Pattern:**
```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import * as apiModule from '../lib/api'

beforeEach(() => vi.clearAllMocks())

it('test name', async () => {
  vi.spyOn(apiModule.api, 'getMe').mockResolvedValue({ id: 1, username: 'test' })
  
  render(<AuthProvider><TestComponent /></AuthProvider>)
  
  await waitFor(() => {
    expect(screen.getByText('Expected')).toBeInTheDocument()
  })
})
```

## API Conventions

**Endpoints:**
- API: `/api/photos`, `/api/photo/{id}`, `/api/me`, `/api/user/{username}`
- Auth: `POST /login`, `POST /register`, `/logout`
- Forms: `POST /upload`, `POST /photo/{id}/delete`, `POST /photo/{id}/edit`

**Auth:** Session-based with HTTP-only cookies. Use `GetCurrentUser(r)` in handlers.

**Request/Response:**
- Auth forms: `application/x-www-form-urlencoded`
- API responses: JSON with snake_case keys
- Pagination: `?page=1`, Filter: `?tags=name1,name2&feed=following`

## Important Notes

- Run `go fmt ./...` and `npm run lint` before commits
- Database: SQLite at `data/nuistagram.db`
- Uploads stored in `static/uploads/`
- Server serves React from `frontend/dist` if exists, else HTML templates
- Path alias `@/` maps to `frontend/src/` in TypeScript
