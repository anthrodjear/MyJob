#!/usr/bin/env bash
# =============================================================================
# scripts/setup-production.sh — Production Environment Setup
# =============================================================================
# One-command production setup: validates prerequisites, generates secrets,
# configures SSL, builds images, runs migrations, and starts all services.
#
# Usage:
#   bash scripts/setup-production.sh                # Full setup
#   bash scripts/setup-production.sh --skip-build   # Skip image builds
#   bash scripts/setup-production.sh --domain example.com --email admin@example.com
#
# Idempotent: safe to run multiple times. Only performs missing steps.
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="${PROJECT_ROOT}/.env.production"
ENV_LOCAL="${PROJECT_ROOT}/.env.production.local"
ENV_TARGET="${PROJECT_ROOT}/.env"

DOMAIN="${DOMAIN:-localhost}"
EMAIL="${EMAIL:-}"
SKIP_BUILD=false
SKIP_MIGRATE=false

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log()    { echo -e "${GREEN}[setup]${NC} $*"; }
warn()   { echo -e "${YELLOW}[setup]${NC} $*"; }
error()  { echo -e "${RED}[setup]${NC} $*" >&2; }
info()   { echo -e "${BLUE}[setup]${NC} $*"; }
step()   { echo -e "\n${CYAN}${BOLD}Step $1: $2${NC}"; }
ok()     { echo -e "${GREEN}  [ok]${NC} $*"; }
fail()   { echo -e "${RED}  [FAIL]${NC} $*"; }

cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        error "Setup failed at line $1 (exit code: $exit_code)"
        error "Check the output above for details."
        error "You can re-run this script - it is idempotent."
    fi
}
trap 'cleanup $LINENO' ERR

generate_secret() {
    local length="${1:-32}"
    openssl rand -hex "$length" 2>/dev/null || head -c "$length" /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c "$length"
}

is_placeholder() {
    local value="$1"
    [[ "$value" == CHANGE_ME_* ]] || [[ "$value" == "" ]] || [[ "$value" == "changeme" ]]
}

update_env_var() {
    local file="$1" key="$2" value="$3"
    if grep -q "^${key}=" "$file" 2>/dev/null; then
        if sed --version 2>/dev/null | grep -q GNU; then
            sed -i "s|^${key}=.*|${key}=${value}|" "$file"
        else
            sed -i '' "s|^${key}=.*|${key}=${value}|" "$file"
        fi
    else
        echo "${key}=${value}" >> "$file"
    fi
}

check_prerequisites() {
    step 1 "Checking prerequisites"
    local missing=()

    if command -v docker &> /dev/null; then
        ok "Docker: $(docker --version | head -1)"
    else
        missing+=("docker")
        fail "Docker is not installed"
    fi

    if docker compose version &> /dev/null; then
        ok "Docker Compose: $(docker compose version --short 2>/dev/null || echo 'v2')"
    elif command -v docker-compose &> /dev/null; then
        ok "Docker Compose (standalone): $(docker-compose --version | head -1)"
    else
        missing+=("docker-compose")
        fail "Docker Compose is not installed"
    fi

    if command -v openssl &> /dev/null; then
        ok "OpenSSL: $(openssl version | head -1)"
    else
        missing+=("openssl")
        fail "OpenSSL is not installed"
    fi

    if command -v curl &> /dev/null; then
        ok "curl: available"
    else
        warn "curl not found - health checks may fail"
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Missing prerequisites: ${missing[*]}"
        exit 1
    fi
    ok "All prerequisites satisfied"
}

setup_environment() {
    step 2 "Setting up environment file"
    local source_env=""
    if [[ -f "$ENV_LOCAL" ]]; then
        source_env="$ENV_LOCAL"
        info "Using local override: ${ENV_LOCAL}"
    elif [[ -f "$ENV_FILE" ]]; then
        source_env="$ENV_FILE"
        info "Using production template: ${ENV_FILE}"
    else
        error "No .env.production file found."
        exit 1
    fi

    if [[ -f "$ENV_TARGET" ]]; then
        warn ".env already exists - not overwriting"
    else
        cp "$source_env" "$ENV_TARGET"
        log "Copied ${source_env} to .env"
    fi

    log "Checking and generating missing secrets..."

    local jwt_secret
    jwt_secret=$(grep "^AUTH_JWT_SECRET=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$jwt_secret"; then
        update_env_var "$ENV_TARGET" "AUTH_JWT_SECRET" "$(generate_secret 32)"
        ok "Generated AUTH_JWT_SECRET"
    else
        ok "AUTH_JWT_SECRET already set"
    fi

    local session_secret
    session_secret=$(grep "^SESSION_SECRET=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$session_secret"; then
        update_env_var "$ENV_TARGET" "SESSION_SECRET" "$(generate_secret 32)"
        ok "Generated SESSION_SECRET"
    else
        ok "SESSION_SECRET already set"
    fi

    local pg_password
    pg_password=$(grep "^POSTGRES_PASSWORD=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$pg_password"; then
        local new_password
        new_password=$(generate_secret 16)
        update_env_var "$ENV_TARGET" "POSTGRES_PASSWORD" "$new_password"
        ok "Generated POSTGRES_PASSWORD"
    else
        ok "POSTGRES_PASSWORD already set"
    fi

    local redis_password
    redis_password=$(grep "^REDIS_PASSWORD=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$redis_password"; then
        local new_password
        new_password=$(generate_secret 16)
        update_env_var "$ENV_TARGET" "REDIS_PASSWORD" "$new_password"
        ok "Generated REDIS_PASSWORD"
    else
        ok "REDIS_PASSWORD already set"
    fi

    local lk_key lk_secret
    lk_key=$(grep "^LIVEKIT_API_KEY=" "$ENV_TARGET" | cut -d'=' -f2-)
    lk_secret=$(grep "^LIVEKIT_API_SECRET=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$lk_key"; then
        update_env_var "$ENV_TARGET" "LIVEKIT_API_KEY" "$(generate_secret 12)"
        ok "Generated LIVEKIT_API_KEY"
    else
        ok "LIVEKIT_API_KEY already set"
    fi
    if is_placeholder "$lk_secret"; then
        update_env_var "$ENV_TARGET" "LIVEKIT_API_SECRET" "$(generate_secret 24)"
        ok "Generated LIVEKIT_API_SECRET"
    else
        ok "LIVEKIT_API_SECRET already set"
    fi

    local grafana_password
    grafana_password=$(grep "^GRAFANA_PASSWORD=" "$ENV_TARGET" | cut -d'=' -f2-)
    if is_placeholder "$grafana_password"; then
        update_env_var "$ENV_TARGET" "GRAFANA_PASSWORD" "$(generate_secret 12)"
        ok "Generated GRAFANA_PASSWORD"
    else
        ok "GRAFANA_PASSWORD already set"
    fi

    update_env_var "$ENV_TARGET" "APP_ENV" "production"
    ok "Environment file ready"
}

setup_password_hash() {
    step 3 "Setting up admin password hash"
    local hash
    hash=$(grep "^AUTH_PASSWORD_HASH=" "$ENV_TARGET" | cut -d'=' -f2-)
    if [[ -n "$hash" && "$hash" != "" ]]; then
        ok "AUTH_PASSWORD_HASH already set"
        return 0
    fi

    warn "AUTH_PASSWORD_HASH is empty."
    warn "Set it with: make hash-password PASSWORD=yourpassword"
    warn "Or run the /setup wizard in the web UI after deployment"
}

setup_ssl() {
    step 4 "Setting up SSL/TLS certificates"
    local domain_value
    domain_value=$(grep "^NGINX_SERVER_NAME=" "$ENV_TARGET" | cut -d'=' -f2-)
    if [[ -n "$domain_value" && "$domain_value" != "your-domain.com" ]]; then
        DOMAIN="$domain_value"
    fi
    if [[ "$DOMAIN" == "localhost" ]]; then
        warn "Domain is localhost - generating self-signed certificate"
    fi
    bash "${SCRIPT_DIR}/setup-ssl.sh" --domain "$DOMAIN" ${EMAIL:+--email "$EMAIL"}
    ok "SSL certificates configured"
}

validate_config() {
    step 5 "Validating Docker Compose configuration"
    cd "$PROJECT_ROOT"
    if docker compose -f docker-compose.yml -f docker-compose.production.yml \
         --env-file .env config --quiet 2>/dev/null; then
        ok "Docker Compose configuration is valid"
    else
        error "Docker Compose configuration is invalid"
        exit 1
    fi
}

build_images() {
    step 6 "Building Docker images"
    if [[ "$SKIP_BUILD" == true ]]; then
        warn "Skipping build (--skip-build flag)"
        return 0
    fi
    cd "$PROJECT_ROOT"
    log "Building all images (this may take several minutes)..."
    docker compose -f docker-compose.yml -f docker-compose.production.yml \
        --env-file .env build --parallel
    ok "All images built successfully"
}

start_infrastructure() {
    step 7 "Starting infrastructure services"
    cd "$PROJECT_ROOT"
    log "Starting PostgreSQL and Redis..."
    docker compose -f docker-compose.yml -f docker-compose.production.yml \
        --env-file .env up -d postgres redis

    log "Waiting for database to be ready..."
    local retries=30
    for i in $(seq 1 "$retries"); do
        if docker compose exec -T postgres pg_isready -U myjob -d myjob >/dev/null 2>&1; then
            ok "PostgreSQL is ready"
            break
        fi
        if [[ $i -eq $retries ]]; then
            error "PostgreSQL did not become ready after ${retries} attempts"
            exit 1
        fi
        sleep 2
    done

    log "Waiting for Redis to be ready..."
    for i in $(seq 1 "$retries"); do
        if docker compose exec -T redis redis-cli ping >/dev/null 2>&1; then
            ok "Redis is ready"
            break
        fi
        if [[ $i -eq $retries ]]; then
            error "Redis did not become ready after ${retries} attempts"
            exit 1
        fi
        sleep 2
    done
}

run_migrations() {
    step 8 "Running database migrations"
    if [[ "$SKIP_MIGRATE" == true ]]; then
        warn "Skipping migrations (--skip-migrate flag)"
        return 0
    fi
    cd "$PROJECT_ROOT"
    log "Starting API service..."
    docker compose -f docker-compose.yml -f docker-compose.production.yml \
        --env-file .env up -d api

    log "Waiting for API to be healthy..."
    local retries=40
    for i in $(seq 1 "$retries"); do
        if docker compose exec -T api wget -qO- http://localhost:8080/health >/dev/null 2>&1; then
            ok "API is healthy"
            break
        fi
        if [[ $i -eq $retries ]]; then
            error "API did not become healthy after ${retries} attempts"
            exit 1
        fi
        sleep 3
    done

    log "Running migrations..."
    if docker compose exec -T api /app/scripts/migrate.sh; then
        ok "Migrations completed"
    else
        warn "Migration script returned non-zero (may be already up-to-date)"
    fi
}

start_services() {
    step 9 "Starting all services"
    cd "$PROJECT_ROOT"
    log "Starting remaining services..."
    docker compose -f docker-compose.yml -f docker-compose.production.yml \
        --env-file .env up -d
    sleep 10
    ok "All services started"
}

run_health_checks() {
    step 10 "Running health checks"
    cd "$PROJECT_ROOT"
    local all_healthy=true

    local services=("api" "worker" "browser-agent" "frontend" "postgres" "redis" "livekit")
    for svc in "${services[@]}"; do
        local status
        status=$(docker compose ps --format json 2>/dev/null | \
                 grep "\"Name\".*${svc}" | head -1 | \
                 grep -o '"State":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        if [[ "$status" == "running" ]]; then
            ok "$svc is running"
        else
            warn "$svc status: ${status:-not found}"
            all_healthy=false
        fi
    done

    log "Checking HTTP endpoints..."
    if curl -sf --max-time 10 http://localhost:8080/health >/dev/null 2>&1; then
        ok "API health endpoint responding"
    else
        warn "API health endpoint not responding on :8080"
    fi
    if curl -sf --max-time 10 http://localhost:3000 >/dev/null 2>&1; then
        ok "Frontend responding on :3000"
    else
        warn "Frontend not responding on :3000"
    fi

    if [[ "$all_healthy" == true ]]; then
        ok "All health checks passed"
    else
        warn "Some services may need attention"
    fi
}

print_summary() {
    local domain_value
    domain_value=$(grep "^NGINX_SERVER_NAME=" "$ENV_TARGET" | cut -d'=' -f2-)
    echo ""
    echo "============================================================"
    echo "  Production Setup Complete!"
    echo "============================================================"
    echo ""
    echo "  Service URLs:"
    echo "    Frontend:      http://localhost:3000"
    echo "    API:           http://localhost:8080"
    echo "    Browser Agent: http://localhost:3001"
    echo "    LiveKit:       ws://localhost:7880"
    echo ""
    echo "  Production URLs (with nginx):"
    echo "    HTTP:          http://${domain_value}"
    echo "    HTTPS:         https://${domain_value}"
    echo ""
    echo "  Useful Commands:"
    echo "    docker compose ps"
    echo "    docker compose logs -f"
    echo "    docker compose down"
    echo "    docker compose up -d"
    echo ""
    echo "  Next Steps:"
    echo "    1. Set AUTH_PASSWORD_HASH: make hash-password PASSWORD=yourpassword"
    echo "    2. Configure LLM API keys in .env"
    echo "    3. Open http://localhost:3000 and complete /setup wizard"
    echo "============================================================"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --domain)    DOMAIN="$2"; shift 2 ;;
        --email)     EMAIL="$2"; shift 2 ;;
        --skip-build)   SKIP_BUILD=true; shift ;;
        --skip-migrate) SKIP_MIGRATE=true; shift ;;
        -h|--help)
            echo "Usage: $(basename "$0") [OPTIONS]"
            echo "  --domain DOMAIN      Domain for SSL"
            echo "  --email EMAIL        Email for Let's Encrypt"
            echo "  --skip-build         Skip Docker builds"
            echo "  --skip-migrate       Skip migrations"
            exit 0 ;;
        *) error "Unknown option: $1"; exit 1 ;;
    esac
done

echo ""
echo "MyJob Production Setup"
echo ""

check_prerequisites
setup_environment
setup_password_hash
setup_ssl
validate_config
build_images
start_infrastructure
run_migrations
start_services
run_health_checks
print_summary
