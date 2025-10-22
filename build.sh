#!/usr/bin/env bash
set -euo pipefail

# ===== Config (override via env or flags) =====
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"     # use arm64 if you ever target Graviton
CGO_ENABLED="${CGO_ENABLED:-0}"
BIN_NAME="${BIN_NAME:-server}"

SERVER_DIR="server"           # where main.go lives
SERVER_STATIC_DIR="server/static"  # <- static lives inside server/
USERS_DIR="users"             # repo-level users directory
BUILD_DIR="build"

usage() {
  cat <<EOF
Usage: $(basename "$0") [--arch amd64|arm64] [--os linux|darwin|windows] [--name BINNAME] [--zip]

Layout produced:
  build/
    ├─ static/        (copy of server/static/)
    ├─ users/         (copy of users/)
    └─ server/
       └─ ${BIN_NAME} (compiled Go binary)

Examples:
  $(basename "$0")                   # build linux/amd64
  $(basename "$0") --arch arm64      # build linux/arm64
  $(basename "$0") --zip             # also create build.tar.gz
EOF
}

ZIP=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch) GOARCH="$2"; shift 2 ;;
    --os)   GOOS="$2";   shift 2 ;;
    --name) BIN_NAME="$2"; shift 2 ;;
    --zip)  ZIP=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: $1"; usage; exit 1 ;;
  esac
done

# ===== Preconditions =====
command -v go >/dev/null 2>&1 || { echo "ERROR: Go toolchain not found"; exit 1; }
[[ -d "$SERVER_DIR" ]] || { echo "ERROR: $SERVER_DIR not found"; exit 1; }
[[ -d "$SERVER_STATIC_DIR" ]] || { echo "WARN: $SERVER_STATIC_DIR not found (static will be empty)"; }
[[ -d "$USERS_DIR" ]] || { echo "WARN: $USERS_DIR not found (users will be empty)"; }

# ===== Clean & create build dirs =====
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/static" "$BUILD_DIR/users" "$BUILD_DIR/server"

# ===== Copy assets =====
if [[ -d "$SERVER_STATIC_DIR" ]]; then
  # copy contents of server/static into build/static
  cp -R "$SERVER_STATIC_DIR"/. "$BUILD_DIR/static/" || true
fi

if [[ -d "$USERS_DIR" ]]; then
  # copy contents of users into build/users
  cp -R "$USERS_DIR"/. "$BUILD_DIR/users/" || true
fi

# ===== Build binary into build/server/server =====
echo "Building ${GOOS}/${GOARCH} (CGO_ENABLED=${CGO_ENABLED}) -> ${BUILD_DIR}/server/${BIN_NAME}"
BIN_EXT=""
[[ "$GOOS" == "windows" ]] && BIN_EXT=".exe"

pushd "$SERVER_DIR" >/dev/null
GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED="$CGO_ENABLED" \
  go build -trimpath -ldflags="-s -w" -o "../${BUILD_DIR}/server/${BIN_NAME}${BIN_EXT}"
popd >/dev/null

# ===== Optional tarball =====
if $ZIP; then
  tar -C "$BUILD_DIR" -czf build.tar.gz .
  echo "Created ./build.tar.gz"
fi

echo "Done. Final layout:"
tree -a "$BUILD_DIR" 2>/dev/null || find "$BUILD_DIR" -print
