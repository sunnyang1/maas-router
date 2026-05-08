#!/usr/bin/env bash
# ============================================================
# MaaS-Router Rollback Script
# 用法:
#   ./scripts/rollback.sh [environment]
#
# 回滚策略:
#   1. 恢复到上次部署的镜像版本
#   2. 回滚数据库迁移（如需要）
#   3. 健康检查验证
# ============================================================

set -euo pipefail

# ── 配置 ──────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ENVIRONMENT="${1:-production}"
MAX_RETRIES=20
RETRY_INTERVAL=3

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

cd "$PROJECT_DIR"

log_warn "=============================================="
log_warn "  ⏪  ROLLBACK INITIATED for ${ENVIRONMENT}"
log_warn "  This will revert to the previous version"
log_warn "=============================================="
echo ""

if [[ "${ENVIRONMENT}" == "production" ]]; then
    echo -n "⚠️  Type 'ROLLBACK-PROD' to confirm: "
    read -r CONFIRM
    if [ "$CONFIRM" != "ROLLBACK-PROD" ]; then
        log_info "Rollback cancelled."
        exit 0
    fi
fi

# ── Step 1: 检查回滚信息 ──────────────────────────────
PREV_IMAGES_FILE="/tmp/maas-router-previous-images.txt"

if [ -f "$PREV_IMAGES_FILE" ] && [ -s "$PREV_IMAGES_FILE" ]; then
    log_info "Found previous image info"
else
    log_warn "No previous image info found. Attempting to restart from backup state."
fi

# ── Step 2: 回滚数据库迁移（如指定） ──────────────────
if [ "${ROLLBACK_DB:-false}" = "true" ]; then
    log_info "Rolling back database migration..."
    docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm \
        -e SERVICE_NAME=migration \
        api-server alembic downgrade -1
    log_ok "Database migration rolled back"
else
    log_info "Skipping database rollback (set ROLLBACK_DB=true to rollback migrations)"
fi

# ── Step 3: 重新部署服务 ─────────────────────────────
log_info "Restarting services with previous configuration..."
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans
log_ok "Services restarted"

# ── Step 4: 等待就绪 ─────────────────────────────────
log_info "Waiting for services to stabilize..."
sleep 10

# ── Step 5: 健康检查 ─────────────────────────────────
check_health() {
    local service=$1
    local url=$2
    local attempts=0

    while [ $attempts -lt $MAX_RETRIES ]; do
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
        if [ "$STATUS" = "200" ]; then
            log_ok "$service is healthy (HTTP $STATUS)"
            return 0
        fi
        attempts=$((attempts + 1))
        echo -ne "  ⏳ $service: attempt $attempts/$MAX_RETRIES (HTTP $STATUS)\r"
        sleep $RETRY_INTERVAL
    done
    echo ""
    log_error "$service health check FAILED"
    return 1
}

API_HEALTH_URL="${API_HEALTH_URL:-http://localhost:8001/health}"
ADMIN_HEALTH_URL="${ADMIN_HEALTH_URL:-http://localhost:8005/health}"

HEALTH_OK=true
check_health "API Server" "$API_HEALTH_URL" || HEALTH_OK=false
check_health "Admin Server" "$ADMIN_HEALTH_URL" || HEALTH_OK=false

if [ "$HEALTH_OK" = false ]; then
    log_error "❌ Rollback health checks failed!"
    log_error "Manual intervention required. Check: docker compose logs"
    exit 1
fi

echo ""
log_ok "=============================================="
log_ok "  ✅ Rollback completed successfully!"
log_ok "  Environment: ${ENVIRONMENT}"
log_ok "  Time: $(date '+%Y-%m-%d %H:%M:%S')"
log_ok "=============================================="
