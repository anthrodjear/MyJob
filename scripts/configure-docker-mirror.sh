#!/usr/bin/env bash
# scripts/configure-docker-mirror.sh
# Configures Docker daemon with registry mirrors to fix timeout issues.
# Run on the Linux server: sudo bash scripts/configure-docker-mirror.sh

set -euo pipefail

DAEMON_JSON="/etc/docker/daemon.json"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[docker]${NC} $*"; }
warn()  { echo -e "${YELLOW}[docker]${NC} $*"; }
error() { echo -e "${RED}[docker]${NC} $*" >&2; }

# Check if running as root
if [[ $EUID -ne 0 ]]; then
  error "This script must be run as root (sudo)"
  exit 1
fi

# Backup existing daemon.json
if [[ -f "$DAEMON_JSON" ]]; then
  warn "Backing up existing $DAEMON_JSON to $DAEMON_JSON.bak"
  cp "$DAEMON_JSON" "$DAEMON_JSON.bak"
fi

# Create daemon.json with mirrors
cat > "$DAEMON_JSON" <<'EOF'
{
  "registry-mirrors": [
    "https://mirror.gcr.io",
    "https://docker.m.daocloud.io"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
EOF

info "Docker daemon configured with registry mirrors:"
info "  - https://mirror.gcr.io (Google Container Registry mirror)"
info "  - https://docker.m.daocloud.io (DaoCloud mirror)"
info ""

# Restart Docker
info "Restarting Docker daemon..."
systemctl restart docker

# Verify
info "Verifying mirrors..."
docker info 2>/dev/null | grep -A5 "Registry Mirrors" || warn "Could not verify mirrors"

info ""
info "Done! Try pulling images again:"
info "  docker compose pull"
info ""
info "If still timing out, check:"
info "  - Network: ping -c 3 mirror.gcr.io"
info "  - DNS: nslookup mirror.gcr.io"
info "  - Firewall: sudo ufw status"
