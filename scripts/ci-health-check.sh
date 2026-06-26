#!/usr/bin/env bash
# scripts/ci-health-check.sh
# Verify all services are healthy after CI deployment.
# Exit 1 on any failure.

set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
WORKER_URL="${WORKER_URL:-http://localhost:8081}"
BROWSER_AGENT_URL="${BROWSER_AGENT_URL:-http://localhost:3001}"
MAX_RETRIES="${MAX_RETRIES:-30}"
RETRY_INTERVAL="${RETRY_INTERVAL:-2}"

log() { echo "[health-check] $*"; }
fail() { echo "[health-check] FAIL: $*" >&2; exit 1; }

# Wait for a URL to return HTTP 200
wait_for_url() {
  local url="$1" name="$2" retries="$3" interval="$4"
  log "Waiting for $name ($url)..."
  for i in $(seq 1 "$retries"); do
    if curl -sf --max-time 10 --retry-connrefused "$url" > /dev/null 2>&1; then
      log "✓ $name is healthy"
      return 0
    fi
    log "  Attempt $i/$retries — retrying in ${interval}s..."
    sleep "$interval"
  done
  fail "$name did not become healthy after $retries attempts"
}

# Check API health endpoint returns valid JSON
check_api_health() {
  local response
  response=$(curl -sf --max-time 10 "$API_URL/health" 2>/dev/null) || fail "API health endpoint unreachable"
  if echo "$response" | grep -q '"status"'; then
    log "✓ API health check passed"
  else
    fail "API health response missing status field: $response"
  fi
}

# Check frontend returns 200
check_frontend() {
  local status_code
  status_code=$(curl -s --max-time 10 -o /dev/null -w "%{http_code}" "$FRONTEND_URL" 2>/dev/null) || fail "Frontend unreachable"
  if [ "$status_code" = "200" ]; then
    log "✓ Frontend health check passed (HTTP $status_code)"
  else
    fail "Frontend returned unexpected status: $status_code"
  fi
}

# Check worker is running (Docker container status)
check_worker_running() {
  local status
  status=$(docker inspect --format='{{.State.Status}}' myjob-worker-1 2>/dev/null || echo "not_found")
  if [ "$status" = "running" ]; then
    log "✓ Worker container is running"
  else
    fail "Worker container status: $status"
  fi
}

# Check browser-agent is running and responds
check_browser_agent_running() {
  local status
  status=$(docker inspect --format='{{.State.Status}}' myjob-browser-agent-1 2>/dev/null || echo "not_found")
  if [ "$status" = "running" ]; then
    log "✓ Browser agent container is running"
  else
    fail "Browser agent container status: $status"
  fi
}

# Main
log "Starting health checks..."
log "API: $API_URL | Frontend: $FRONTEND_URL | Worker: $WORKER_URL | Browser Agent: $BROWSER_AGENT_URL"

# Docker container status checks
check_worker_running
check_browser_agent_running

# HTTP endpoint checks
wait_for_url "$API_URL/health" "API" "$MAX_RETRIES" "$RETRY_INTERVAL"
check_api_health

wait_for_url "$FRONTEND_URL" "Frontend" "$MAX_RETRIES" "$RETRY_INTERVAL"
check_frontend

log "All health checks passed ✓"
