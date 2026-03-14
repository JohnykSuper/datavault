FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем только необходимое для сборки
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /datavault ./cmd/datavault

# ── Production image ──────────────────────────────────────────────────────
FROM alpine:3.19 AS production

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /datavault /datavault
EXPOSE 8080
ENTRYPOINT ["/datavault"]
