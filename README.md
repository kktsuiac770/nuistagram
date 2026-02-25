# NUIstagram

A photo sharing application with a Go backend and React frontend.

## Tech Stack

- **Backend**: Go 1.25, SQLite
- **Frontend**: React, TypeScript, Vite, Tailwind CSS

## Prerequisites

- Go 1.25+
- Node.js 18+

## Quick Start

### Backend

```bash
go build -o nuistagram ./cmd/server
./nuistagram
```

The server runs on port 8080.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The dev server runs on port 5173 with proxy to backend.

## Access

Open http://localhost:8080 in your browser.
