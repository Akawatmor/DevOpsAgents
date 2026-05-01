# syntax=docker/dockerfile:1.7

# ── Builder ─────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=local

# Cache deps layer
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server .

# ── Runtime (distroless, ~2MB base) ─────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/server /app/server

# SQLite จะถูกเขียนที่ /data (mount เป็น PVC)
ENV DB_PATH=/data/auth.db \
    PORT=8080

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]