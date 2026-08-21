#!/usr/bin/env bash
# =============================================================================
# scripts/deploy-setup.sh — Interactive Deployment Setup
# =============================================================================
# Interactive wizard that fills .env files and deploys via Docker Compose or
# Kubernetes (Helm/kustomize). Generates all secrets, validates config,
# and handles the full deployment lifecycle.
#
# Usage:
#   bash scripts/deploy-setup.sh                    # Interactive (default)
#   bash scripts/deploy-setup.sh --method docker    # Skip method selection
#   bash scripts/deploy-setup.sh --method k8s       # Skip method selection
#   bash scripts/deploy-setup.sh --non-interactive   # Use all defaults
#   bash scripts/deploy-setup.sh --dry-run          # Preview without deploying
# =============================================================================

set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'
NC='\033[0m'

log()    { echo -e "${GREEN}[setup]${NC} $*"; }
warn()   { echo -e "${YELLOW}[setup]${NC} $*"; }
error()  { echo -e "${RED}[setup]${NC} $*" >&2; }
info()   { echo -e "${BLUE}[setup]${NC} $*"; }
step()   { echo -e "\n${CYAN}${BOLD}── Step $1: $2 ──${NC}"; }
ok()     { echo -e "${GREEN}  ✓${NC} $*"; }
fail()   { echo -e "${RED}  ✗${NC} $*"; }
die()    { error "$@"; exit 1; }
header() {
    echo ""
    echo -e "${CYAN}${BOLD}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}${BOLD}║           MyJob — Deployment Setup Wizard               ║${NC}"
    echo -e "${CYAN}${BOLD}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# ── Globals ───────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

DEPLOY_METHOD=""
DOMAIN="localhost"
NAMESPACE="myjob"
EMAIL=""
DRY_RUN=false
NON_INTERACTIVE=false
SKIP_BUILD=false
SKIP_MIGRATE=false
NAMESPACE_SET=false
ADMIN_PASSWORD=""
ENV_PROFILE="production"  # development | production
TEMP_FILES=()  # Track temp files for cleanup on failure

# ── Usage ─────────────────────────────────────────────────────────────────────
usage() {
    cat <<'EOF'
Usage: bash scripts/deploy-setup.sh [OPTIONS]

Interactive deployment wizard for MyJob.

Options:
  --method docker|k8s       Deployment method (skips interactive prompt)
  --profile dev|production  Environment profile (default: production)
  --domain DOMAIN           Domain name for ingress/nginx (default: localhost)
  --namespace NS            Kubernetes namespace (default: myjob)
  --email EMAIL             Email for Let's Encrypt SSL
  --dry-run                 Preview config without deploying
  --non-interactive         Use all defaults, no prompts
  --skip-build              Skip Docker image builds
  --skip-migrate            Skip database migrations
  -h, --help                Show this help
EOF
}

# ── Argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --method)        DEPLOY_METHOD="$2"; shift 2 ;;
        --domain)        DOMAIN="$2"; shift 2 ;;
        --namespace)     NAMESPACE="$2"; NAMESPACE_SET=true; shift 2 ;;
        --email)         EMAIL="$2"; shift 2 ;;
        --password)      die "Do not use --password on command line (visible in ps). Use the interactive prompt." ;;
        --profile)       ENV_PROFILE="$2"; shift 2 ;;
        --dry-run)       DRY_RUN=true; shift ;;
        --non-interactive) NON_INTERACTIVE=true; shift ;;
        --skip-build)    SKIP_BUILD=true; shift ;;
        --skip-migrate)  SKIP_MIGRATE=true; shift ;;
        -h|--help)       usage; exit 0 ;;
        *) error "Unknown option: $1"; exit 1 ;;
    esac
done

# Validate --method if provided
if [[ -n "$DEPLOY_METHOD" ]]; then
    case "$DEPLOY_METHOD" in
        docker|k8s-helm|k8s-kustomize) ;;
        *) die "Invalid --method '$DEPLOY_METHOD'. Must be: docker, k8s-helm, or k8s-kustomize" ;;
    esac
fi


# ── Utility ───────────────────────────────────────────────────────────────────
generate_secret() {
    local length="${1:-32}"
    local secret
    secret=$(openssl rand -hex "$length" 2>/dev/null) || \
        secret=$(head -c "$length" /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c "$((length * 2))")
    if [[ -z "$secret" ]]; then
        die "Failed to generate secret (length=$length). Check openssl or /dev/urandom."
    fi
    echo "$secret"
}

generate_password() {
    local length="${1:-24}"
    local pass
    pass=$(openssl rand -base64 "$length" 2>/dev/null | tr -d '/+=' | head -c "$length")
    if [[ -z "$pass" ]]; then
        die "Failed to generate password (length=$length). Check openssl or /dev/urandom."
    fi
    echo "$pass"
}

# ── Validation ────────────────────────────────────────────────────────────────
validate_namespace() {
    local ns="$1"
    if [[ ! "$ns" =~ ^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$ ]]; then
        die "Invalid namespace '$ns'. Must be RFC 1123 compliant: lowercase alphanumeric and hyphens, 63 chars max."
    fi
    if [[ ${#ns} -gt 63 ]]; then
        die "Namespace '$ns' too long (${#ns} chars). Max 63 characters."
    fi
}

validate_domain() {
    local domain="$1"
    # Allow IP addresses
    if [[ "$domain" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
    fi
    # Allow localhost
    if [[ "$domain" == "localhost" ]]; then
        return 0
    fi
    # Basic domain format check
    if [[ ! "$domain" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]; then
        warn "Domain '$domain' doesn't look like a valid domain or IP. Proceeding anyway."
    fi
}

validate_password_strength() {
    local pw="$1" name="$2" min_len="${3:-8}"
    if [[ ${#pw} -lt $min_len ]]; then
        die "$name is too short (${#pw} chars). Minimum $min_len characters."
    fi
}

is_placeholder() {
    local value="$1"
    [[ "$value" == CHANGE_ME_* ]] || [[ "$value" == "" ]] || [[ "$value" == "changeme" ]] || \
    [[ "$value" == "devkey" ]] || [[ "$value" == "devsecret" ]]
}

update_env_var() {
    local file="$1" key="$2" value="$3"
    if [[ ! -f "$file" ]]; then
        die "update_env_var: file '$file' does not exist"
    fi
    local tmpfile
    tmpfile=$(mktemp "${file}.XXXXXX")
    TEMP_FILES+=("$tmpfile")
    if grep -q "^${key}=" "$file" 2>/dev/null; then
        awk -v key="$key" -v value="$value" '
            $0 ~ "^" key "=" { print key "=" value; next }
            { print }
        ' "$file" > "$tmpfile" && mv "$tmpfile" "$file"
    else
        echo "${key}=${value}" >> "$file"
        rm -f "$tmpfile"
    fi
}

# Prompt for input with default
prompt() {
    local var_name="$1" prompt_text="$2" default_value="$3" secret="${4:-false}"
    if [[ "$NON_INTERACTIVE" == true ]]; then
        printf -v "$var_name" '%s' "$default_value"
        return
    fi
    if [[ "$secret" == true ]]; then
        echo -ne "${BOLD}$prompt_text${NC} [${DIM}${default_value:0:4}****${NC}]: "
        read -rs input
        echo ""
    else
        echo -ne "${BOLD}$prompt_text${NC} [${DIM}$default_value${NC}]: "
        read -r input
    fi
    printf -v "$var_name" '%s' "${input:-$default_value}"
}

# Prompt with options
prompt_choice() {
    local var_name="$1" prompt_text="$2" options="$3" default_value="$4"
    if [[ "$NON_INTERACTIVE" == true ]]; then
        printf -v "$var_name" '%s' "$default_value"
        return
    fi
    echo -e "${BOLD}$prompt_text${NC}"
    local i=1
    IFS=',' read -ra opts <<< "$options"
    for opt in "${opts[@]}"; do
        if [[ "$opt" == "$default_value" ]]; then
            echo -e "  ${GREEN}$i${NC}) $opt ${GREEN}(default)${NC}"
        else
            echo -e "  ${DIM}$i${NC}) $opt"
        fi
        ((i++))
    done
    echo -ne "  Enter [1-${#opts[@]}]: "
    read -r choice
    if [[ -z "$choice" ]] || ! [[ "$choice" =~ ^[0-9]+$ ]] || (( choice < 1 || choice > ${#opts[@]} )); then
        printf -v "$var_name" '%s' "$default_value"
    else
        printf -v "$var_name" '%s' "${opts[$((choice-1))]}"
    fi
}

confirm() {
    local prompt_text="$1" default="${2:-y}"
    if [[ "$NON_INTERACTIVE" == true ]]; then
        return 0
    fi
    local hint="Y/n"
    [[ "$default" == "n" ]] && hint="y/N"
    echo -ne "${BOLD}$prompt_text${NC} [$hint]: "
    read -r answer
    answer="${answer:-$default}"
    [[ "$answer" =~ ^[Yy] ]]
}

# ── Step: Method Selection ───────────────────────────────────────────────────
select_method() {
    if [[ -n "$DEPLOY_METHOD" ]]; then
        ok "Deployment method: $DEPLOY_METHOD"
        return
    fi

    step 1 "Select Deployment Method"
    prompt_choice DEPLOY_METHOD "How do you want to deploy?" \
        "docker,k8s-helm,k8s-kustomize" "docker"

    case "$DEPLOY_METHOD" in
        docker)         log "Selected: Docker Compose" ;;
        k8s-helm)       log "Selected: Kubernetes (Helm)" ;;
        k8s-kustomize)  log "Selected: Kubernetes (Kustomize)" ;;
    esac
}

# ── Step: Environment Profile ────────────────────────────────────────────────
select_profile() {
    case "$ENV_PROFILE" in
        dev|development) ENV_PROFILE="development" ;;
        prod|production) ENV_PROFILE="production" ;;
        *) die "Unknown profile '$ENV_PROFILE'. Use: dev or production" ;;
    esac
    ok "Environment: $ENV_PROFILE"
}

# ── Step: Collect Secrets ────────────────────────────────────────────────────
collect_secrets() {
    step 2 "Configure Secrets & Credentials"

    echo -e "${DIM}Press Enter to accept defaults shown in brackets.${NC}\n"

    # --- Admin Password ---
    if [[ -z "$ADMIN_PASSWORD" ]]; then
        prompt ADMIN_PASSWORD "  Admin password" "$(generate_password 16)"
    fi
    validate_password_strength "$ADMIN_PASSWORD" "Admin password" 8
    info "Admin password: set (${#ADMIN_PASSWORD} chars)"

    # --- Database ---
    echo ""
    info "── Database ──"
    prompt POSTGRES_PASSWORD "  PostgreSQL password" "$(generate_password 16)"

    # --- Redis ---
    info "── Redis ──"
    prompt REDIS_PASSWORD "  Redis password" "$(generate_password 16)"

    # --- Auth ---
    echo ""
    info "── Authentication ──"
    AUTH_JWT_SECRET=$(generate_secret 32)
    ok "AUTH_JWT_SECRET: [set] (auto-generated)"
    SESSION_SECRET=$(generate_secret 32)
    ok "SESSION_SECRET: [set] (auto-generated)"

    # Generate bcrypt hash
    info "Generating password hash..."
    AUTH_PASSWORD_HASH=""
    if command -v htpasswd &>/dev/null; then
        AUTH_PASSWORD_HASH=$(htpasswd -nbBC 10 "" "$ADMIN_PASSWORD" 2>/dev/null | cut -d: -f2)
    elif [[ -f "$PROJECT_ROOT/backend/go.mod" ]]; then
        local hash_output
        hash_output=$(cd "$PROJECT_ROOT" && go run scripts/hash_password.go "$ADMIN_PASSWORD" 2>&1) || hash_output=""
        if [[ "$hash_output" =~ ^\$2 ]]; then
            AUTH_PASSWORD_HASH="$hash_output"
        else
            AUTH_PASSWORD_HASH=""
        fi
    fi
    if [[ -n "$AUTH_PASSWORD_HASH" ]]; then
        ok "Password hash: [set]"
    else
        if [[ "$ENV_PROFILE" == "production" ]]; then
            die "Cannot generate password hash in production. Install htpasswd or ensure Go is available."
        else
            warn "Could not generate password hash. Set via: make hash-password PASSWORD=yourpassword"
        fi
    fi

    # --- LLM API Keys ---
    echo ""
    info "── LLM API Keys (optional — Ollama works without these) ──"
    prompt OPENAI_API_KEY "  OpenAI API key" ""
    prompt ANTHROPIC_API_KEY "  Anthropic API key" ""

    # --- LiveKit ---
    echo ""
    info "── LiveKit (Voice Interview Coach) ──"
    LIVEKIT_API_KEY=$(generate_secret 12)
    LIVEKIT_API_SECRET=$(generate_secret 24)
    ok "LIVEKIT_API_KEY: [set] (auto-generated)"
    ok "LIVEKIT_API_SECRET: [set] (auto-generated)"

    # --- Microsoft Graph ---
    echo ""
    info "── Microsoft Graph (Email Sync — optional) ──"
    prompt MS_TENANT_ID "  Tenant ID" ""
    prompt MS_CLIENT_ID "  Client ID" ""
    prompt MS_CLIENT_SECRET "  Client secret" ""

    # --- Grafana ---
    echo ""
    info "── Monitoring ──"
    prompt GRAFANA_PASSWORD "  Grafana admin password" "$(generate_password 12)"
}

# ── Step: Collect Domain / Ingress ───────────────────────────────────────────
collect_domain() {
    step 3 "Domain & Networking"

    if [[ "$DEPLOY_METHOD" == "docker" ]]; then
        prompt DOMAIN "  Domain (or IP for local)" "localhost"
        prompt CORS_ORIGINS "  CORS origins" "http://localhost:3000,http://127.0.0.1:3000"
    else
        prompt DOMAIN "  Domain for ingress" "192.168.2.102"
        if [[ "$NAMESPACE_SET" == false ]]; then
            prompt NAMESPACE "  Kubernetes namespace" "myjob"
        fi
        prompt CORS_ORIGINS "  CORS origins" "http://${DOMAIN},https://${DOMAIN}"
    fi

    # Validate inputs
    if [[ "$DEPLOY_METHOD" != "docker" ]]; then
        validate_namespace "$NAMESPACE"
    fi
    validate_domain "$DOMAIN"

    # Sanitize domain for use in CORS_ORIGINS
    if [[ "$DOMAIN" =~ [^a-zA-Z0-9.\-:] ]]; then
        die "Domain contains invalid characters: $DOMAIN"
    fi
}

# ── Step: Docker Compose Deployment ──────────────────────────────────────────
deploy_docker() {
    step 4 "Docker Compose — Generating .env"

    if ! docker info >/dev/null 2>&1; then
        die "Cannot access Docker. Run with sudo or add your user to the docker group."
    fi

    local env_file="$PROJECT_ROOT/.env"
    local env_template="$PROJECT_ROOT/.env.production"

    if [[ "$ENV_PROFILE" == "development" ]]; then
        env_template="$PROJECT_ROOT/.env.example"
    fi

    # Copy template
    if [[ -f "$env_file" ]]; then
        if confirm "  Overwrite existing .env?" "n"; then
            local bak="$env_file.bak.$(date +%s)"
            cp "$env_file" "$bak"
            chmod 600 "$bak"
            ok "  Backed up existing .env"
        else
            warn "  Using existing .env as base"
        fi
    fi

    if [[ ! -f "$env_file" ]]; then
        if [[ ! -f "$env_template" ]]; then
            die "Template '$env_template' not found. Ensure .env.production or .env.example exists."
        fi
        cp "$env_template" "$env_file"
        chmod 600 "$env_file"  # Set permissions immediately
        ok "  Created .env from template"
    fi

    # Write all values
    info "  Writing environment variables..."

    update_env_var "$env_file" "POSTGRES_PASSWORD" "$POSTGRES_PASSWORD"
    update_env_var "$env_file" "REDIS_PASSWORD" "$REDIS_PASSWORD"
    update_env_var "$env_file" "AUTH_JWT_SECRET" "$AUTH_JWT_SECRET"
    update_env_var "$env_file" "SESSION_SECRET" "$SESSION_SECRET"
    update_env_var "$env_file" "LIVEKIT_API_KEY" "$LIVEKIT_API_KEY"
    update_env_var "$env_file" "LIVEKIT_API_SECRET" "$LIVEKIT_API_SECRET"
    update_env_var "$env_file" "GRAFANA_PASSWORD" "$GRAFANA_PASSWORD"
    update_env_var "$env_file" "APP_ENV" "$ENV_PROFILE"
    update_env_var "$env_file" "NGINX_SERVER_NAME" "$DOMAIN"
    update_env_var "$env_file" "CORS_ORIGINS" "$CORS_ORIGINS"

    # API keys — only update if user provided them
    [[ -n "$OPENAI_API_KEY" ]] && update_env_var "$env_file" "OPENAI_API_KEY" "$OPENAI_API_KEY"
    [[ -n "$ANTHROPIC_API_KEY" ]] && update_env_var "$env_file" "ANTHROPIC_API_KEY" "$ANTHROPIC_API_KEY"
    [[ -n "$MS_TENANT_ID" ]] && update_env_var "$env_file" "MS_TENANT_ID" "$MS_TENANT_ID"
    [[ -n "$MS_CLIENT_ID" ]] && update_env_var "$env_file" "MS_CLIENT_ID" "$MS_CLIENT_ID"
    [[ -n "$MS_CLIENT_SECRET" ]] && update_env_var "$env_file" "MS_CLIENT_SECRET" "$MS_CLIENT_SECRET"

    # Store bcrypt hash separately (docker-compose $-escaping issue)
    if [[ -n "$AUTH_PASSWORD_HASH" ]]; then
        echo "$AUTH_PASSWORD_HASH" > "$PROJECT_ROOT/.env.auth"
        chmod 600 "$PROJECT_ROOT/.env.auth"
        ok "  .env.auth created (bcrypt hash)"
    fi

    # Production-specific
    if [[ "$ENV_PROFILE" == "production" ]]; then
        update_env_var "$env_file" "LIVEKIT_FLAGS" ""
        update_env_var "$env_file" "OLLAMA_MODEL" "nemotron-3-super:cloud"
        update_env_var "$env_file" "OLLAMA_EMBED_MODEL" "granite-embedding:latest"
        update_env_var "$env_file" "QUEUE_CONCURRENCY" "10"
        update_env_var "$env_file" "RATE_LIMIT_RPM" "120"
        update_env_var "$env_file" "RATE_LIMIT_BURST" "20"
    fi

    ok "  .env file ready"
    chmod 600 "$env_file"

    # --- Deploy ---
    step 5 "Docker Compose — Deploy"

    if [[ "$DRY_RUN" == true ]]; then
        warn "  DRY RUN — showing command that would execute:"
        if [[ "$ENV_PROFILE" == "production" ]]; then
            echo "    docker compose -f docker-compose.yml -f docker-compose.production.yml --env-file .env up -d"
        else
            echo "    docker compose --env-file .env up -d"
        fi
        return
    fi

    if ! confirm "  Start services now?" "y"; then
        info "  Skipping startup. Run manually:"
        if [[ "$ENV_PROFILE" == "production" ]]; then
            echo "    docker compose -f docker-compose.yml -f docker-compose.production.yml --env-file .env up -d"
        else
            echo "    docker compose --env-file .env up -d"
        fi
        return
    fi

    cd "$PROJECT_ROOT"

    local compose_args=()
    if [[ "$ENV_PROFILE" == "production" ]]; then
        compose_args+=("-f" "docker-compose.yml" "-f" "docker-compose.production.yml")
    fi
    compose_args+=("--env-file" ".env")

    # Build images (unless skipped)
    if [[ "$SKIP_BUILD" != true ]]; then
        log "  Building Docker images..."
        if ! docker compose "${compose_args[@]}" build 2>&1; then
            error "Docker build failed. Check logs: docker compose build --progress=plain"
            exit 1
        fi
        ok "  Images built"
    else
        info "  Skipping image build (--skip-build)"
    fi

    # Start infra first
    log "  Starting infrastructure (PostgreSQL, Redis)..."
    if ! docker compose "${compose_args[@]}" up -d postgres redis livekit; then
        error "Failed to start infrastructure services."
        exit 1
    fi

    log "  Waiting for database..."
    local retries=30
    for i in $(seq 1 "$retries"); do
        if docker compose "${compose_args[@]}" exec -T postgres pg_isready -U myjob -d myjob >/dev/null 2>&1; then
            ok "  PostgreSQL is ready"
            break
        fi
        [[ $i -eq $retries ]] && { fail "  PostgreSQL not ready after ${retries} attempts"; exit 1; }
        sleep 2
    done

    # Start API
    log "  Starting API..."
    if ! docker compose "${compose_args[@]}" up -d api; then
        error "Failed to start API service."
        exit 1
    fi
    log "  Waiting for API health check..."
    retries=40
    for i in $(seq 1 "$retries"); do
        if docker compose "${compose_args[@]}" exec -T api wget -qO- http://localhost:8080/health >/dev/null 2>&1; then
            ok "  API is healthy"
            break
        fi
        [[ $i -eq $retries ]] && { fail "  API not healthy after ${retries} attempts"; exit 1; }
        sleep 3
    done

    # Run migrations (unless skipped)
    if [[ "$SKIP_MIGRATE" != true ]]; then
        log "  Running database migrations..."
        if docker compose "${compose_args[@]}" exec -T api /app/scripts/migrate.sh 2>/dev/null; then
            ok "  Migrations complete"
        else
            warn "  Migration script returned non-zero (may be already up-to-date)"
        fi
    else
        info "  Skipping migrations (--skip-migrate)"
    fi

    # Start remaining
    log "  Starting remaining services..."
    if ! docker compose "${compose_args[@]}" up -d; then
        error "Failed to start remaining services."
        exit 1
    fi

    sleep 5

    # Verify critical containers are running
    log "  Verifying critical containers..."
    local critical_services=("api" "postgres" "redis")
    for svc in "${critical_services[@]}"; do
        local state
        state=$(docker compose "${compose_args[@]}" ps --format json 2>/dev/null | \
                grep "\"Name\".*${svc}" | head -1 | \
                grep -o '"State":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        if [[ "$state" != "running" ]]; then
            warn "  $svc: ${state:-not found} — check logs with: docker compose logs $svc"
        fi
    done

    ok "  All services started"

    # Health check
    step 6 "Health Checks"
    local services=("api" "worker" "browser-agent" "frontend" "postgres" "redis" "livekit")
    for svc in "${services[@]}"; do
        local state
        state=$(docker compose "${compose_args[@]}" ps --format json 2>/dev/null | \
                grep "\"Name\".*${svc}" | head -1 | \
                grep -o '"State":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
        if [[ "$state" == "running" ]]; then
            ok "  $svc: running"
        else
            warn "  $svc: ${state:-not found}"
        fi
    done

    # SSL setup for production
    if [[ "$ENV_PROFILE" == "production" && -n "$EMAIL" ]]; then
        info "  Running SSL setup for $DOMAIN with email $EMAIL..."
        if ! bash "${SCRIPT_DIR}/setup-ssl.sh" --domain "$DOMAIN" --email "$EMAIL" 2>&1; then
            warn "SSL setup failed. You may need to install cert-manager first."
        fi
    fi

    echo ""
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}  Docker deployment complete!${NC}"
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo "  URLs:"
    echo "    Frontend:      http://${DOMAIN}:3000"
    echo "    API:           http://${DOMAIN}:8080"
    echo "    Browser Agent: http://${DOMAIN}:3001"
    echo ""
    echo "  Commands:"
    echo "    docker compose logs -f         # Tail logs"
    echo "    docker compose ps              # Status"
    echo "    docker compose down            # Stop"
    echo ""
}

# ── Step: Kubernetes (Helm) Deployment ────────────────────────────────────────
deploy_k8s_helm() {
    step 4 "Kubernetes (Helm) — Creating Secrets"

    if [[ "$DRY_RUN" == true ]]; then
        warn "  DRY RUN — showing commands that would execute:"
        print_k8s_commands
        return
    fi

    # Check kubectl
    if ! command -v kubectl &>/dev/null; then
        error "kubectl not found. Install it first."
        exit 1
    fi
    if ! command -v helm &>/dev/null; then
        error "helm not found. Install it first."
        exit 1
    fi

    # Create namespace
    log "  Creating namespace: $NAMESPACE"
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    ok "  Namespace ready"

    # Create secrets
    log "  Creating database secret..."
    kubectl create secret generic myjob-database \
        --namespace="$NAMESPACE" \
        --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-database secret"

    log "  Creating auth secret..."
    kubectl create secret generic myjob-auth \
        --namespace="$NAMESPACE" \
        --from-literal=AUTH_JWT_SECRET="$AUTH_JWT_SECRET" \
        --from-literal=AUTH_PASSWORD_HASH="${AUTH_PASSWORD_HASH:-}" \
        --from-literal=SESSION_SECRET="$SESSION_SECRET" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-auth secret"

    log "  Creating redis secret..."
    kubectl create secret generic myjob-redis \
        --namespace="$NAMESPACE" \
        --from-literal=REDIS_PASSWORD="$REDIS_PASSWORD" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-redis secret"

    log "  Creating livekit secret..."
    kubectl create secret generic myjob-livekit \
        --namespace="$NAMESPACE" \
        --from-literal=LIVEKIT_API_KEY="$LIVEKIT_API_KEY" \
        --from-literal=LIVEKIT_API_SECRET="$LIVEKIT_API_SECRET" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-livekit secret"

    log "  Creating LLM secret..."
    kubectl create secret generic myjob-llm \
        --namespace="$NAMESPACE" \
        --from-literal=OPENAI_API_KEY="${OPENAI_API_KEY:-}" \
        --from-literal=ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-llm secret"

    log "  Creating Microsoft Graph secret..."
    kubectl create secret generic myjob-microsoft \
        --namespace="$NAMESPACE" \
        --from-literal=MS_TENANT_ID="${MS_TENANT_ID:-}" \
        --from-literal=MS_CLIENT_ID="${MS_CLIENT_ID:-}" \
        --from-literal=MS_CLIENT_SECRET="${MS_CLIENT_SECRET:-}" \
        --dry-run=client -o yaml | kubectl apply -f -
    ok "  myjob-microsoft secret"

    ok "  All secrets created"

    # Generate Helm values override
    step 5 "Kubernetes (Helm) — Generating Values Override"
    local values_file="$PROJECT_ROOT/k8s/helm/myjob/values-${NAMESPACE}.yaml"
    cat > "$values_file" <<VALUES_EOF
# Auto-generated by deploy-setup.sh — $(date '+%Y-%m-%dT%H:%M:%S%z')
# Namespace: $NAMESPACE
# Domain: $DOMAIN

namespace: $NAMESPACE

ingress:
  enabled: true
  hosts:
    - host: $DOMAIN
      paths:
        - path: /
          pathType: Prefix
          service: frontend
          port: 3000
        - path: /api
          pathType: Prefix
          service: api
          port: 8080
        - path: /livekit
          pathType: Prefix
          service: livekit
          port: 7880
  tls:
    - secretName: myjob-tls
      hosts:
        - $DOMAIN

secrets:
  database:
    secretName: myjob-database
  auth:
    secretName: myjob-auth
  livekit:
    secretName: myjob-livekit
  redis:
    secretName: myjob-redis
  llm:
    secretName: myjob-llm
  microsoft:
    secretName: myjob-microsoft

monitoring:
  enabled: true
  grafana:
    adminPassword: "$GRAFANA_PASSWORD"
VALUES_EOF
    chmod 600 "$values_file"
    ok "  Values override: $values_file"

    # Deploy
    step 6 "Kubernetes (Helm) — Deploying"

    if ! confirm "  Install/upgrade Helm release now?" "y"; then
        info "  Skipping deployment. Run manually:"
        echo "    helm upgrade --install myjob-$NAMESPACE ./k8s/helm/myjob \\"
        echo "      --namespace $NAMESPACE --create-namespace \\"
        echo "      --values k8s/helm/myjob/values-${NAMESPACE}.yaml"
        return
    fi

    log "  Installing Helm release..."
    helm upgrade --install "myjob-$NAMESPACE" ./k8s/helm/myjob \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --values "k8s/helm/myjob/values-${NAMESPACE}.yaml" \
        --atomic --timeout 300s

    ok "  Helm release installed"

    if [[ -n "$EMAIL" ]]; then
        info "  SSL tip: For Let's Encrypt, install cert-manager and use --email $EMAIL"
    fi

    step 7 "Kubernetes — Health Check"
    log "  Checking pods..."
    kubectl get pods -n "$NAMESPACE" -o wide
    echo ""
    log "  Checking ingress..."
    kubectl get ingress -n "$NAMESPACE"

    echo ""
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}  Kubernetes (Helm) deployment complete!${NC}"
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo "  Namespace: $NAMESPACE"
    echo "  Domain:    $DOMAIN"
    echo ""
    echo "  Commands:"
    echo "    kubectl get pods -n $NAMESPACE"
    echo "    kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=myjob --tail=100 -f"
    echo "    helm rollback myjob-$NAMESPACE -n $NAMESPACE"
    echo ""
}

# ── Step: Kubernetes (Kustomize) Deployment ───────────────────────────────────
deploy_k8s_kustomize() {
    step 4 "Kubernetes (Kustomize) — Generating Config"

    local overlay_dir="$PROJECT_ROOT/k8s/kustomize/overlays/$NAMESPACE"
    mkdir -p "$overlay_dir"

    # Generate secrets file (sealed or plain)
    local secrets_file="$overlay_dir/secrets.env"
    cat > "$secrets_file" <<SECRETS_EOF
# Auto-generated by deploy-setup.sh — $(date '+%Y-%m-%dT%H:%M:%S%z')
# DO NOT COMMIT — add to .gitignore
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
REDIS_PASSWORD=$REDIS_PASSWORD
AUTH_JWT_SECRET=$AUTH_JWT_SECRET
AUTH_PASSWORD_HASH=${AUTH_PASSWORD_HASH:-}
SESSION_SECRET=$SESSION_SECRET
LIVEKIT_API_KEY=$LIVEKIT_API_KEY
LIVEKIT_API_SECRET=$LIVEKIT_API_SECRET
OPENAI_API_KEY=${OPENAI_API_KEY:-}
ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-}
MS_TENANT_ID=${MS_TENANT_ID:-}
MS_CLIENT_ID=${MS_CLIENT_ID:-}
MS_CLIENT_SECRET=${MS_CLIENT_SECRET:-}
GRAFANA_PASSWORD=$GRAFANA_PASSWORD
SECRETS_EOF
    chmod 600 "$secrets_file"
    ok "  Secrets file: $secrets_file"

    # Ensure secrets.env is gitignored
    local gitignore="$PROJECT_ROOT/.gitignore"
    if [[ -f "$gitignore" ]] && ! grep -qF "secrets.env" "$gitignore"; then
        echo "secrets.env" >> "$gitignore"
        ok "  Added secrets.env to .gitignore"
    fi

    # Generate kustomization.yaml overlay if missing
    local kustom_file="$overlay_dir/kustomization.yaml"
    if [[ ! -f "$kustom_file" ]]; then
        cat > "$kustom_file" <<KUSTOM_EOF
# Kustomize overlay for: $NAMESPACE
# Generated by deploy-setup.sh
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: $NAMESPACE

resources:
  - ../../base

patches:
  - target:
      kind: Deployment
      name: api
    patch: |
      - op: replace
        path: /spec/template/spec/containers/0/env/0/value
        value: "$NAMESPACE"

commonLabels:
  environment: $NAMESPACE
KUSTOM_EOF
        ok "  Created: $kustom_file"
    fi

    # Deploy
    step 5 "Kubernetes (Kustomize) — Deploying"

    if [[ "$DRY_RUN" == true ]]; then
        warn "  DRY RUN — showing manifest:"
        kubectl kustomize "$overlay_dir"
        return
    fi

    if ! command -v kubectl &>/dev/null; then
        error "kubectl not found. Install it first."
        exit 1
    fi

    log "  Creating namespace..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    log "  Creating secrets from $secrets_file..."
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
        if ! kubectl create secret generic "myjob-${key,,//_/-}" \
            --namespace="$NAMESPACE" \
            --from-literal="$key=$value" \
            --dry-run=client -o yaml | kubectl apply -f -; then
            error "Failed to create secret for key: $key"
            exit 1
        fi
    done < "$secrets_file"
    ok "  Secrets applied"

    if ! confirm "  Apply kustomize manifest now?" "y"; then
        info "  Skipping. Run manually:"
        echo "    kubectl apply -k $overlay_dir"
        return
    fi

    log "  Applying kustomize overlay..."
    if ! kubectl apply -k "$overlay_dir"; then
        error "Kustomize apply failed. Debug: kubectl apply -k $overlay_dir --dry-run=client"
        exit 1
    fi

    ok "  Manifest applied"
    log "  Checking pods..."
    kubectl get pods -n "$NAMESPACE"

    echo ""
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}  Kubernetes (Kustomize) deployment complete!${NC}"
    echo -e "${GREEN}${BOLD}══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo "  Namespace: $NAMESPACE"
    echo ""
    echo "  Commands:"
    echo "    kubectl get pods -n $NAMESPACE"
    echo "    kubectl apply -k $overlay_dir    # Re-apply"
    echo ""
}

print_k8s_commands() {
    echo "  # Create namespace"
    echo "  kubectl create namespace $NAMESPACE"
    echo ""
    echo "  # Create secrets"
    echo "  kubectl create secret generic myjob-database -n $NAMESPACE \\"
    echo "    --from-literal=POSTGRES_PASSWORD=[set]"
    echo "  kubectl create secret generic myjob-auth -n $NAMESPACE \\"
    echo "    --from-literal=AUTH_JWT_SECRET=[set] \\"
    echo "    --from-literal=AUTH_PASSWORD_HASH=[set] \\"
    echo "    --from-literal=SESSION_SECRET=[set]"
    echo "  kubectl create secret generic myjob-redis -n $NAMESPACE \\"
    echo "    --from-literal=REDIS_PASSWORD=[set]"
    echo "  kubectl create secret generic myjob-livekit -n $NAMESPACE \\"
    echo "    --from-literal=LIVEKIT_API_KEY=[set] \\"
    echo "    --from-literal=LIVEKIT_API_SECRET=[set]"
    echo "  kubectl create secret generic myjob-llm -n $NAMESPACE \\"
    echo "    --from-literal=OPENAI_API_KEY=[set] \\"
    echo "    --from-literal=ANTHROPIC_API_KEY=[set]"
    echo ""
    echo "  # Deploy with Helm"
    echo "  helm upgrade --install myjob-$NAMESPACE ./k8s/helm/myjob \\"
    echo "    --namespace $NAMESPACE --create-namespace \\"
    echo "    --set api.image.tag=latest \\"
    echo "    --set worker.image.tag=latest \\"
    echo "    --set frontend.image.tag=latest \\"
    echo "    --set browserAgent.image.tag=latest"
    echo ""
    echo "  # Or deploy with Kustomize"
    echo "  kubectl apply -k k8s/kustomize/overlays/$NAMESPACE"
}

# ── Summary ───────────────────────────────────────────────────────────────────
print_summary() {
    echo ""
    echo -e "${CYAN}${BOLD}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}${BOLD}║                    Setup Complete!                       ║${NC}"
    echo -e "${CYAN}${BOLD}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "  Environment:   $ENV_PROFILE"
    echo "  Method:        $DEPLOY_METHOD"
    echo "  Domain:        $DOMAIN"
    [[ "$DEPLOY_METHOD" != "docker" ]] && echo "  Namespace:     $NAMESPACE"
    echo ""
    echo "  Generated files:"
    [[ "$DEPLOY_METHOD" == "docker" ]] && echo "    .env              — Docker Compose environment"
    [[ "$DEPLOY_METHOD" == "docker" ]] && echo "    .env.auth         — Bcrypt password hash (separate for \$-escaping)"
    [[ "$DEPLOY_METHOD" == "k8s-helm" ]] && echo "    k8s/helm/myjob/values-${NAMESPACE}.yaml — Helm values override"
    [[ "$DEPLOY_METHOD" == "k8s-kustomize" ]] && echo "    k8s/kustomize/overlays/${NAMESPACE}/ — Kustomize overlay + secrets"
    echo ""
    echo "  Secrets generated:"
    echo "    AUTH_JWT_SECRET       [set]"
    echo "    SESSION_SECRET        [set]"
    echo "    POSTGRES_PASSWORD     [set]"
    echo "    REDIS_PASSWORD        [set]"
    echo "    LIVEKIT_API_KEY       [set]"
    echo "    LIVEKIT_API_SECRET    [set]"
    echo ""
    echo "  Next steps:"
    if [[ "$DEPLOY_METHOD" == "docker" ]]; then
        echo "    1. Review .env:  cat .env"
        echo "    2. Set password: make hash-password PASSWORD=yourpassword"
        echo "    3. Open browser: http://${DOMAIN}:3000"
    else
        echo "    1. Review secrets:  kubectl get secrets -n $NAMESPACE"
        echo "    2. Check pods:      kubectl get pods -n $NAMESPACE"
        echo "    3. Check ingress:   kubectl get ingress -n $NAMESPACE"
    fi
    echo ""
    echo "  IMPORTANT: Add .env.auth and any secrets files to .gitignore"
    echo ""
}

# ── Main ──────────────────────────────────────────────────────────────────────
main() {
    header
    select_method
    select_profile
    collect_secrets
    collect_domain

    case "$DEPLOY_METHOD" in
        docker)        deploy_docker ;;
        k8s-helm)      deploy_k8s_helm ;;
        k8s-kustomize) deploy_k8s_kustomize ;;
    esac

    print_summary
}

# Cleanup trap
cleanup() {
    local exit_code=$?
    for f in "${TEMP_FILES[@]}"; do
        [[ -f "$f" ]] && rm -f "$f"
    done
    if [[ $exit_code -ne 0 ]]; then
        error "Setup failed at line $LINENO (exit code $exit_code)."
        error "Re-run with --dry-run to preview."
    fi
}
trap cleanup EXIT

main "$@"
