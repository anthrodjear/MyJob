#!/usr/bin/env bash
# =============================================================================
# scripts/setup-ssl.sh — SSL/TLS Certificate Setup
# =============================================================================
# Generates self-signed certificates for development or obtains Let's Encrypt
# certificates for production. Creates DH parameters for enhanced security.
#
# Usage:
#   bash scripts/setup-ssl.sh                          # Self-signed (dev)
#   bash scripts/setup-ssl.sh --domain example.com     # Self-signed for domain
#   bash scripts/setup-ssl.sh --letsencrypt             # Let's Encrypt (prod)
#   bash scripts/setup-ssl.sh --letsencrypt --domain example.com --email admin@example.com
#
# Requirements:
#   - openssl (required)
#   - certbot (only for Let's Encrypt)
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CERT_DIR="${PROJECT_ROOT}/certs"
LIVE_DIR="${CERT_DIR}/live"
DH_PARAM_FILE="${CERT_DIR}/dhparam.pem"

DOMAIN="${DOMAIN:-localhost}"
EMAIL="${EMAIL:-}"
USE_LETSENCRYPT=false
FORCE_RENEW=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ---------------------------------------------------------------------------
# Helper Functions
# ---------------------------------------------------------------------------
log()    { echo -e "${GREEN}[ssl]${NC} $*"; }
warn()   { echo -e "${YELLOW}[ssl]${NC} $*"; }
error()  { echo -e "${RED}[ssl]${NC} $*" >&2; }
info()   { echo -e "${BLUE}[ssl]${NC} $*"; }

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --domain DOMAIN        Domain name for the certificate (default: localhost)
  --email EMAIL          Email for Let's Encrypt notifications
  --letsencrypt          Use Let's Encrypt instead of self-signed
  --force                Force certificate renewal
  -h, --help             Show this help message

Examples:
  $(basename "$0")                                        # Self-signed for localhost
  $(basename "$0") --domain myapp.example.com             # Self-signed for domain
  $(basename "$0") --letsencrypt --domain myapp.example.com --email admin@example.com
EOF
    exit 0
}

check_openssl() {
    if ! command -v openssl &> /dev/null; then
        error "openssl is not installed."
        error "Install it with:"
        error "  Ubuntu/Debian: sudo apt-get install openssl"
        error "  macOS:         brew install openssl"
        error "  CentOS/RHEL:   sudo yum install openssl"
        exit 1
    fi
    log "openssl found: $(openssl version)"
}

check_certbot() {
    if ! command -v certbot &> /dev/null; then
        error "certbot is not installed."
        error "Install it with:"
        error "  Ubuntu/Debian: sudo apt-get install certbot"
        error "  macOS:         brew install certbot"
        error "  CentOS/RHEL:   sudo yum install certbot"
        exit 1
    fi
    log "certbot found: $(certbot --version 2>&1)"
}

create_directories() {
    log "Creating certificate directory structure..."
    mkdir -p "${LIVE_DIR}/${DOMAIN}"
    log "Directory structure created at ${CERT_DIR}"
}

# ---------------------------------------------------------------------------
# Self-Signed Certificate (Development / Testing)
# ---------------------------------------------------------------------------
generate_self_signed() {
    local domain_dir="${LIVE_DIR}/${DOMAIN}"
    local cert_file="${domain_dir}/fullchain.pem"
    local key_file="${domain_dir}/privkey.pem"
    local csr_file="${domain_dir}/csr.pem"
    local ext_file="${domain_dir}/ext.cnf"

    # Skip if certificates already exist (unless --force)
    if [[ -f "$cert_file" && -f "$key_file" && "$FORCE_RENEW" == false ]]; then
        warn "Certificates already exist at ${domain_dir}"
        warn "Use --force to regenerate"
        return 0
    fi

    log "Generating self-signed certificate for: ${DOMAIN}"

    # Create OpenSSL config with SAN (Subject Alternative Name)
    cat > "$ext_file" <<EXTEOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
x509_extensions    = v3_req

[dn]
C  = US
ST = State
L  = City
O  = MyJob
OU = Development
CN = ${DOMAIN}

[v3_req]
basicConstraints = CA:TRUE
keyUsage = digitalSignature, keyEncipherment, keyCertSign
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${DOMAIN}
DNS.2 = *.${DOMAIN}
DNS.3 = localhost
IP.1  = 127.0.0.1
IP.2  = ::1
EXTEOF

    # Generate private key (4096-bit for production-grade security)
    openssl genrsa -out "$key_file" 4096 2>/dev/null

    # Generate self-signed certificate (valid 365 days)
    openssl req -new -x509 \
        -key "$key_file" \
        -out "$cert_file" \
        -days 365 \
        -config "$ext_file" \
        2>/dev/null

    # Clean up temporary files
    rm -f "$csr_file" "$ext_file"

    # Set permissions
    chmod 600 "$key_file"
    chmod 644 "$cert_file"

    log "Self-signed certificate generated:"
    log "  Certificate: ${cert_file}"
    log "  Private key: ${key_file}"
    log "  Valid for:   365 days"

    # Display certificate info
    info "Certificate details:"
    openssl x509 -in "$cert_file" -noout -subject -issuer -dates 2>/dev/null | sed 's/^/    /'
}

# ---------------------------------------------------------------------------
# Let's Encrypt Certificate (Production)
# ---------------------------------------------------------------------------
generate_letsencrypt() {
    local domain_dir="${LIVE_DIR}/${DOMAIN}"
    local cert_file="${domain_dir}/fullchain.pem"
    local key_file="${domain_dir}/privkey.pem"

    check_certbot

    if [[ -z "$EMAIL" ]]; then
        error "Email is required for Let's Encrypt."
        error "Usage: $(basename "$0") --letsencrypt --domain example.com --email admin@example.com"
        exit 1
    fi

    log "Obtaining Let's Encrypt certificate for: ${DOMAIN}"

    # Check if certbot can obtain the certificate (dry run first)
    if ! certbot certonly \
        --standalone \
        --non-interactive \
        --agree-tos \
        --email "$EMAIL" \
        --domain "$DOMAIN" \
        --dry-run 2>/dev/null; then
        warn "Dry run failed. Ensure port 80 is open and DNS points to this server."
        warn "Falling back to self-signed certificate..."
        generate_self_signed
        return
    fi

    # Obtain the certificate
    certbot certonly \
        --standalone \
        --non-interactive \
        --agree-tos \
        --email "$EMAIL" \
        --domain "$DOMAIN"

    # Copy certificates to project directory
    mkdir -p "$domain_dir"
    cp "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" "$cert_file"
    cp "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" "$key_file"

    # Set permissions
    chmod 600 "$key_file"
    chmod 644 "$cert_file"

    log "Let's Encrypt certificate obtained:"
    log "  Certificate: ${cert_file}"
    log "  Private key: ${key_file}"

    # Set up auto-renewal
    setup_auto_renewal
}

setup_auto_renewal() {
    log "Setting up certificate auto-renewal..."

    local renewal_script="${SCRIPT_DIR}/renew-ssl.sh"
    cat > "$renewal_script" <<'RENEWEOF'
#!/usr/bin/env bash
# Auto-renewal script for Let's Encrypt certificates
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOMAIN="${1:-localhost}"

# Renew certificate
certbot renew --quiet --deploy-hook "cp /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ${PROJECT_ROOT}/certs/live/${DOMAIN}/fullchain.pem && cp /etc/letsencrypt/live/${DOMAIN}/privkey.pem ${PROJECT_ROOT}/certs/live/${DOMAIN}/privkey.pem && chmod 600 ${PROJECT_ROOT}/certs/live/${DOMAIN}/privkey.pem && docker compose restart nginx"

echo "[$(date)] SSL certificate renewal check completed"
RENEWEOF
    chmod +x "$renewal_script"

    # Add cron job for auto-renewal (twice daily)
    local cron_cmd="0 0,12 * * * ${renewal_script} ${DOMAIN} >> /var/log/ssl-renewal.log 2>&1"
    if ! crontab -l 2>/dev/null | grep -q "$renewal_script"; then
        (crontab -l 2>/dev/null; echo "$cron_cmd") | crontab -
        log "Auto-renewal cron job installed (twice daily)"
    else
        warn "Auto-renewal cron job already exists"
    fi
}

# ---------------------------------------------------------------------------
# DH Parameters
# ---------------------------------------------------------------------------
generate_dhparam() {
    if [[ -f "$DH_PARAM_FILE" && "$FORCE_RENEW" == false ]]; then
        warn "DH parameters already exist at ${DH_PARAM_FILE}"
        warn "Use --force to regenerate"
        return 0
    fi

    log "Generating Diffie-Hellman parameters (2048-bit)..."

    # 2048-bit for faster startup; use 4096 for high-security environments
    openssl dhparam -out "$DH_PARAM_FILE" 2048 2>/dev/null
    chmod 600 "$DH_PARAM_FILE"

    log "DH parameters generated: ${DH_PARAM_FILE}"
    info "For higher security, regenerate with 4096-bit (slower):"
    info "  openssl dhparam -out ${DH_PARAM_FILE} 4096"
}

# ---------------------------------------------------------------------------
# Production Instructions
# ---------------------------------------------------------------------------
print_instructions() {
    local domain_dir="${LIVE_DIR}/${DOMAIN}"
    local cert_file="${domain_dir}/fullchain.pem"
    local key_file="${domain_dir}/privkey.pem"

    cat <<EOF

${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
${GREEN}  SSL/TLS Setup Complete${NC}
${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}

  ${BLUE}Certificate Files:${NC}
    Full chain:  ${cert_file}
    Private key: ${key_file}
    DH params:   ${DH_PARAM_FILE}

  ${BLUE}File Permissions:${NC}
    Private key: $(stat -c '%a' "$key_file" 2>/dev/null || stat -f '%Lp' "$key_file" 2>/dev/null)
    Certificate: $(stat -c '%a' "$cert_file" 2>/dev/null || stat -f '%Lp' "$cert_file" 2>/dev/null)

  ${BLUE}Production Deployment Steps:${NC}

    1. Update .env.production with your domain:
       NGINX_SERVER_NAME=${DOMAIN}

    2. Update nginx virtual host config:
       server_name ${DOMAIN};

    3. Start services with production config:
       docker compose -f docker-compose.yml -f docker-compose.production.yml \\
         --env-file .env.production up -d

    4. Verify SSL is working:
       curl -I https://${DOMAIN}
       # Should show: HTTP/2 200 with strict-transport-security header

  ${BLUE}Certificate Renewal (Let's Encrypt only):${NC}
    - Auto-renewal is configured via cron (twice daily)
    - Manual renewal: certbot renew
    - Check renewal status: certbot certificates

  ${YELLOW}⚠  Security Notes:${NC}
    - Never commit private keys to version control
    - The certs/ directory should be in .gitignore
    - For production, use 4096-bit DH params: openssl dhparam -out ${DH_PARAM_FILE} 4096
    - Rotate certificates before expiry (Let's Encrypt: 90 days)

${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
EOF
}

# ---------------------------------------------------------------------------
# Parse Arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --letsencrypt)
            USE_LETSENCRYPT=true
            shift
            ;;
        --force)
            FORCE_RENEW=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            error "Unknown option: $1"
            usage
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
log "SSL/TLS Certificate Setup"
log "Domain: ${DOMAIN}"
log "Mode: $([ "$USE_LETSENCRYPT" = true ] && echo 'Let'\''s Encrypt' || echo 'Self-Signed')"

check_openssl
create_directories
generate_dhparam

if [[ "$USE_LETSENCRYPT" == true ]]; then
    generate_letsencrypt
else
    generate_self_signed
fi

print_instructions
