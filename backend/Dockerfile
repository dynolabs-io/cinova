# ---- Builder ----
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build both binaries with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/ingestion ./cmd/ingestion

# ---- Final image ----
FROM alpine:3.20

# Install CA certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 65534 nogroup && \
    adduser -u 65534 -G nogroup -D -H nobody 2>/dev/null || true

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/ingestion /usr/local/bin/ingestion
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 65534

EXPOSE 8080

CMD ["/usr/local/bin/api"]
