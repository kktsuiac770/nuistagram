# ─── Stage 1: Build frontend ──────────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder

WORKDIR /build/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ─── Stage 2: Build Go binary ─────────────────────────────────────────────────
FROM golang:1.24-alpine AS backend-builder

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o nuistagram ./cmd/server

# ─── Stage 3: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3.21

# ca-certificates: required for Cloudinary HTTPS uploads
# tzdata: correct timezone handling
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /build/nuistagram ./nuistagram
COPY --from=backend-builder /build/migrations ./migrations
COPY --from=frontend-builder /build/frontend/dist ./frontend/dist
COPY backend/static ./static

EXPOSE 8080

ENTRYPOINT ["./nuistagram"]
