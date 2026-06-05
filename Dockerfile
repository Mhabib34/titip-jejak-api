# ── Stage 1: Builder ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# Install git (dibutuhkan beberapa go module)
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files dulu — layer ini di-cache selama go.mod/go.sum tidak berubah
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary — CGO disabled agar bisa jalan di alpine minimal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

# ── Stage 2: Runner ───────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates dibutuhkan untuk HTTPS (Supabase, Cloudinary, Resend)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy hanya binary dari stage builder
COPY --from=builder /app/server .

# Render inject PORT otomatis via env
EXPOSE 8080

CMD ["./server"]