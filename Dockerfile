FROM golang:1.23-alpine AS builder

WORKDIR /app

# Download dependencies first (separate layer — cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux go build -o /datavault ./cmd/datavault

# ── Production image ──────────────────────────────────────────────────────
FROM alpine:3.19 AS production

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /datavault /datavault
EXPOSE 8080
ENTRYPOINT ["/datavault"]
