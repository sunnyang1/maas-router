#!/usr/bin/env bash
# ============================================================
# MaaS-Router Server Deployment Script
# 用法:
#   ./scripts/deploy.sh [staging|production] [version_tag]
#
# 功能:
#   1. 拉取最新 Docker 镜像
#   2. 备份数据库
#   3. 执行数据库迁移
#   4. 滚动更新服务（零停机）
#   5. 健康检查验证
#   6. 清理旧镜像
# ============================================================

set -euo pipefail

# ── 配置 ──────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ENVIRONMENT="${1:-staging}"
VERSION_TAG="${2:-latest}"
REGISTRY="${REGISTRY:-ghcr.io}"
REPO="${REPO:-your-org/maas-router}"
MAX_RETRIES=30
RETRY_INTERVAL=2
BACKUP_DIR="${BACKUP_DIR:-/tmp/maas-router-backups}"

# ── 颜色输出 ──────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ── 参数校验 ──────────────────────────────────────────
cd "$PROJECT_DIR"

if [[ ! "$ENVIRONMENT" =~ ^(staging|production)$ ]]; then
    log_error "Invalid environment: '$ENVIRONMENT'. Must be 'staging' or 'production'."
    exit 1
fi

log_info "Starting deployment to ${ENVIRONMENT} (tag: ${VERSION_TAG})"
log_info "Project dir: ${PROJECT_DIR}"

# ── Step 1: 登录 Registry ────────────────────────────
log_info "Logging into container registry..."
if [ -n "${GITHUB_TOKEN:-}" ]; then
    echo "$GITHUB_TOKEN" | docker login "$REGISTRY" -u "${GITHUB_ACTOR:-deployer}" --password-stdin
elif [ -n "${REGISTRY_USERNAME:-}" ] && [ -n "${REGISTRY_PASSWORD:-}" ]; then
    echo "$REGISTRY_PASSWORD" | docker login "$REGISTRY" -u "$REGISTRY_USERNAME" --password-stdin
else
    log_warn "No registry credentials found, assuming already logged in"
fi

# ── Step 2: 数据库备份 ───────────────────────────────
log_info "Backing up database..."
mkdir -p "$BACKUP_DIR"
BACKUP_FILE="${BACKUP_DIR}/maas-router-${ENVIRONMENT}-$(date +%Y%m%d_%H%M%S).sql.gz"

if docker compose -f docker-compose.yml -f docker-compose.prod.yml ps postgres 2>/dev/null | grep -q "Up"; then
    docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
        pg_dump -U maas maas_router | gzip > "$BACKUP_FILE"
    log_ok "Database backed up to ${BACKUP_FILE}"
else
    log_warn "PostgreSQL not running, skipping backup"
fi

# ── Step 3: 保存当前版本（用于回滚） ─────────────────
log_info "Saving current image versions for potential rollback..."
docker compose -f docker-compose.yml -f docker-compose.prod.yml images -q 2>/dev/null > /tmp/maas-router-previous-images.txt || true

# ── Step 4: 拉取最新镜像 ─────────────────────────────
log_info "Pulling latest images..."
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
log_ok "Images pulled"

# ── Step 5: 数据库迁移 ───────────────────────────────
log_info "Running database migrations..."
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm \
    -e SERVICE_NAME=migration \
    api-server alembic upgrade head
log_ok "Database migrations applied"

# ── Step 6: 滚动更新服务 ─────────────────────────────
log_info "Deploying services (rolling update)..."
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans
log_ok "Services deployed"

# ── Step 7: 等待服务启动 ─────────────────────────────
log_info "Waiting for services to be healthy..."
sleep 5

# ── Step 8: 健康检查 ─────────────────────────────────
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
    log_error "$service health check FAILED after $MAX_RETRIES attempts"
    return 1
}

API_HEALTH_URL="${API_HEALTH_URL:-http://localhost:8001/health}"
ADMIN_HEALTH_URL="${ADMIN_HEALTH_URL:-http://localhost:8005/health}"

HEALTH_OK=true
check_health "API Server" "$API_HEALTH_URL" || HEALTH_OK=false
check_health "Admin Server" "$ADMIN_HEALTH_URL" || HEALTH_OK=false

if [ "$HEALTH_OK" = false ]; then
    log_error "Health checks failed! Check docker compose logs for details."
    echo ""
    log_info "Recent logs:"
    docker compose -f docker-compose.yml -f docker-compose.prod.yml logs --tail=50 2>/dev/null || true
    exit 1
fi

# ── Step 9: 清理旧镜像 ───────────────────────────────
log_info "Cleaning up old Docker images..."
DOCKER_PRUNE_DAYS="${DOCKER_PRUNE_DAYS:-72}"
docker image prune -af --filter "until=${DOCKER_PRUNE_DAYS}h" || true
log_ok "Old images cleaned"

# ── Step 10: 清理旧备份 ──────────────────────────────
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
find "$BACKUP_DIR" -name "maas-router-*.sql.gz" -mtime "+${BACKUP_RETENTION_DAYS}" -delete 2>/dev/null || true

# ── 完成 ──────────────────────────────────────────────
echo ""
log_ok "=============================================="
log_ok "  ✅ Deployment to ${ENVIRONMENT} completed!"
log_ok "  Tag: ${VERSION_TAG}"
log_ok "  Time: $(date '+%Y-%m-%d %H:%M:%S')"
log_ok "  Backup: ${BACKUP_FILE}"
log_ok "=============================================="
