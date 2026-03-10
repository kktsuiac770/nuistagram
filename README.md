# NUIstagram

A self-hosted photo-sharing platform inspired by Instagram. Users can upload photos, follow each other, like and comment on posts, organize content with tags ("Nuis"), and receive real-time notifications — all through a clean React UI backed by a Go API.

## Features

- **Photo uploads** — automatic compression to max 1920px wide, 85% JPEG quality; 400×400 thumbnails generated on upload
- **EXIF metadata** — date and metadata extracted from photos automatically
- **Tags ("Nuis")** — categorize photos with custom tags and browse by tag
- **Albums** — group photos into named albums
- **Social graph** — follow/unfollow users; feeds filtered by following or all users
- **Likes & comments** — interact on any photo
- **User profiles** — custom bio, avatar (auto-cropped to 200×200), and photo gallery
- **Notifications** — in-app notifications for follows, likes, and comments
- **User search** — find users by username
- **Photo export** — download a ZIP of all your uploaded photos
- **Dark mode** — full dark/light theme support
- **Pluggable storage** — store images locally or on Cloudinary

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, net/http, SQLite (go-sqlite3) |
| Frontend | React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4 |
| Auth | JWT (HS256 access tokens, opaque refresh tokens) |
| Image processing | disintegration/imaging, dsoprea/go-exif |
| Observability | slog structured logging, Prometheus metrics, custom tracing |
| Testing (BE) | testify (mocks + assertions) |
| Testing (FE) | Vitest 4, React Testing Library 16 |

## Prerequisites

- Go 1.24+
- Node.js 18+

## Quick Start

The easiest way is to use Make from the repo root:

```bash
make dev
```

This starts both the backend (port 8080) and frontend dev server (port 5173) concurrently, and creates required directories (`data/`, `static/uploads/`) if they don't exist.

Then open [http://localhost:8080](http://localhost:8080) in your browser.

### Manual start

**Backend** (from `backend/`):

```bash
cp config.example.yaml config.yaml   # configure JWT_SECRET and other settings
go run ./cmd/server
```

**Frontend** (from `frontend/`):

```bash
npm install
npm run dev
```

## Configuration

Create `backend/config.yaml` from the example file. Key environment variables:

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | — | **Required.** Secret key for signing JWT tokens |
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `data/nuistagram.db` | SQLite database path |
| `STORAGE_PROVIDER` | `local` | `local` or `cloudinary` |
| `STORAGE_UPLOAD_DIR` | `static/uploads` | Upload directory (local storage) |
| `CLOUDINARY_CLOUD_NAME` | — | Cloudinary credentials (if using Cloudinary) |
| `ENV` | `development` | Set to `production` for JSON logging |

## Authentication

NUIstagram uses JWT-based authentication:

- **Access token** — short-lived JWT (15 min), sent as `Authorization: Bearer <token>`
- **Refresh token** — opaque 32-byte token (7-day TTL), used to get a new token pair via `POST /api/refresh`
- Tokens are stored in `localStorage` on the frontend and auto-refreshed on expiry

## Image Storage

- **Local** (default): images saved to `static/uploads/`; served directly by the Go server
- **Cloudinary**: images uploaded via signed API; full HTTPS URLs stored in the database

## Running Tests

**Backend:**

```bash
cd backend
go test ./...                          # all tests
go test ./internal/server -v           # handler unit tests (verbose)
go test -v ./... -race -coverprofile=coverage.out  # race detector + coverage
```

**Frontend:**

```bash
cd frontend
npm run test:run       # single pass
npm run test:coverage  # coverage report
```

## Project Structure

```
nuistagram/
├── backend/
│   ├── cmd/server/main.go       # Binary entry point
│   └── internal/
│       ├── database/            # SQLite init, schema, auto-migrations
│       ├── jwt/                 # JWT manager (access + refresh tokens)
│       ├── models/              # Domain structs (User, Photo, Comment, …)
│       ├── repository/          # Data access interfaces + SQLite implementations
│       ├── server/              # HTTP handlers and routes
│       └── storage/             # Pluggable image storage (local / Cloudinary)
└── frontend/
    └── src/
        ├── pages/               # Route-level components
        ├── components/          # Shared UI components
        ├── contexts/            # AuthContext, ThemeContext
        ├── hooks/               # Custom React hooks
        └── lib/api.ts           # Centralised API client with auto token refresh
```
