
# ---------- Build Stage ----------
FROM golang:1.26.5-alpine AS builder

WORKDIR /app

# Copy dependency manifests first for better layer caching
COPY go.mod go.sum ./

RUN go mod download

# Copy application source
COPY . .

# Build a statically linked Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /goqueue \
    ./cmd/api

# ---------- Runtime Stage ----------
FROM alpine:3.22

WORKDIR /app

# Install runtime certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

# Copy only the compiled application
COPY --from=builder /goqueue ./goqueue

# Run as a non-root user
USER app

EXPOSE 8080

ENTRYPOINT ["./goqueue"]
