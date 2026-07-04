#!/usr/bin/env bash
# scripts/setup-env.sh
# Generates .env from .env.example with a valid bcrypt password hash.
# Run once before first build: bash scripts/setup-env.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env"
ENV_EXAMPLE="$PROJECT_ROOT/.env.example"

DEFAULT_PASSWORD="${1:-admin}"

# --- colours (safe for piped output) ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

info()  { echo -e "${GREEN}[setup]${NC} $*"; }
warn()  { echo -e "${YELLOW}[setup]${NC} $*"; }
error() { echo -e "${RED}[setup]${NC} $*" >&2; }

# ── 1. Check .env.example exists ──────────────────────────────────────────────
if [[ ! -f "$ENV_EXAMPLE" ]]; then
  error ".env.example not found at $ENV_EXAMPLE"
  exit 1
fi

# ── 2. Generate bcrypt hash ──────────────────────────────────────────────────
generate_hash() {
  if command -v htpasswd &>/dev/null; then
    # -B = bcrypt, -n = no newline, -C 10 = cost 10
    htpasswd -nbBC 10 "" "$DEFAULT_PASSWORD" 2>/dev/null | cut -d: -f2
  elif command -v python3 &>/dev/null; then
    python3 -c "
import hashlib, base64, os
# Fallback: use Go binary if available
import subprocess
result = subprocess.run(
    ['go', 'run', '$SCRIPT_DIR/hash_password.go', '$DEFAULT_PASSWORD'],
    capture_output=True, text=True,
    cwd='$PROJECT_ROOT/backend'
)
if result.returncode == 0:
    print(result.stdout.strip())
else:
    raise SystemExit('Failed to generate hash')
" 2>/dev/null
  else
    error "Neither htpasswd nor python3 found. Install one to generate the password hash."
    exit 1
  fi
}

info "Generating bcrypt hash for default password..."
HASH=$(generate_hash)

if [[ -z "$HASH" ]]; then
  error "Failed to generate password hash"
  exit 1
fi

info "Hash generated: ${HASH:0:7}...${HASH: -4}"

# ── 3. Build .env from .env.example ──────────────────────────────────────────
if [[ -f "$ENV_FILE" ]]; then
  warn ".env already exists — backing up to .env.bak"
  cp "$ENV_FILE" "$ENV_FILE.bak"
fi

# Copy example as base
cp "$ENV_EXAMPLE" "$ENV_FILE"

# Generate JWT secret if empty
JWT_SECRET=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | base64 | tr -d '\n/+=' | head -c 64)

# Replace empty AUTH_JWT_SECRET with generated secret
# NOTE: AUTH_PASSWORD_HASH is intentionally NOT written to .env because
# bcrypt hashes contain $ characters that docker-compose interpolates.
# Instead, make start/build exports it as a shell environment variable.
# Use awk to avoid $ interpretation issues in sed
awk -v secret="$JWT_SECRET" '
  /^AUTH_JWT_SECRET=$/    { print "AUTH_JWT_SECRET=" secret; next }
  { print }
' "$ENV_FILE" > "$ENV_FILE.tmp" && mv "$ENV_FILE.tmp" "$ENV_FILE"

# Store the hash in a separate file for Make to consume
echo "$HASH" > "$PROJECT_ROOT/.env.auth"

info ".env created at $ENV_FILE"
info ".env.auth created (bcrypt hash, not in .env to avoid \$ interpolation)"
info "  AUTH_PASSWORD_HASH = ${HASH:0:7}... (bcrypt, cost 10)"
info "  AUTH_JWT_SECRET    = ${JWT_SECRET:0:7}..."
info ""
info "Default login password: $DEFAULT_PASSWORD"
info ""
info "To start services:  make start"
