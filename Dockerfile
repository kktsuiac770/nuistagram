# Backend image — Go binary only
# Frontend is served by a separate nginx container (see Dockerfile.frontend)

FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o nuistagram ./cmd/server

# ─── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.21

# ca-certificates: required for Cloudinary HTTPS uploads
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/nuistagram ./nuistagram
COPY --from=builder /build/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./nuistagram"]
