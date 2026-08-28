#!/usr/bin/env bash
# Cross-compiles Kessel into dist/ with the embedded UI.
# The frontend must be built first (web/dist populated); this script builds it
# if node_modules is present, otherwise assumes web/dist already exists.
set -euo pipefail

VERSION="${VERSION:-dev}"
LDFLAGS="-s -w -X github.com/t0mer/kessel/internal/version.Version=${VERSION}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ -f web/package.json ]; then
  echo ">> building frontend"
  (cd web && npm ci && npm run build)
fi

mkdir -p dist

# OS/ARCH targets. Format: "os/arch[/goarm]".
targets=(
  "linux/amd64" "linux/arm64" "linux/arm/7" "linux/arm/6" "linux/386"
  "darwin/amd64" "darwin/arm64"
  "windows/amd64" "windows/arm64"
)

for t in "${targets[@]}"; do
  IFS='/' read -r GOOS GOARCH GOARM <<<"$t"
  out="dist/kessel_${GOOS}_${GOARCH}${GOARM:+v$GOARM}"
  [ "$GOOS" = "windows" ] && out="${out}.exe"
  echo ">> building $out"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" GOARM="${GOARM:-}" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out" ./cmd/kessel
done

echo ">> done: $(ls dist | wc -l) artifacts in dist/"
