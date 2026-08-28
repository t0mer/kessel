# syntax=docker/dockerfile:1

# Stage 1: build the frontend once on the native build platform.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build   # Vite outDir -> web/dist

# Stage 2: cross-compile the Go binary with the embedded UI.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=local \
    CGO_ENABLED=0
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Real built assets replace the tracked placeholder before the go:embed build.
COPY --from=frontend /app/web/dist ./web/dist
ARG VERSION=docker
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} go build \
    -trimpath -ldflags="-s -w -X github.com/t0mer/kessel/internal/version.Version=${VERSION}" \
    -o /out/kessel ./cmd/kessel
# Pre-create the data dir so the scratch image ships a writable volume mount point.
RUN mkdir -p /out/data

# Stage 3: minimal scratch runtime.
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/kessel /kessel
COPY --from=builder /out/data /data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/kessel", "--version"]
ENTRYPOINT ["/kessel"]
CMD ["serve", "--data-dir", "/data"]
