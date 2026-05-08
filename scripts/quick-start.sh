#!/bin/bash

# MaaS-Router 一键安装脚本
# 支持 Linux 和 macOS
# 使用方法: curl -fsSL https://raw.githubusercontent.com/your-repo/maas-router/main/scripts/quick-start.sh | bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 打印 banner
print_banner() {
    echo ""
    echo "=========================================="
    echo "   MaaS-Router 一键安装脚本"
    echo "=========================================="
    echo ""
}

# 检查系统要求
check_requirements() {
    log_info "检查系统要求..."
    
    # 检查操作系统
    OS="$(uname -s)"
    case "${OS}" in
        Linux*)     PLATFORM=Linux;;
        Darwin*)    PLATFORM=Mac;;
        *)          PLATFORM="UNKNOWN:${OS}"
    esac
    
    log_info "检测到操作系统: ${PLATFORM}"
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        log_info "安装指南: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    # 检查 Docker Compose
    if command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    elif docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    else
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_success "Docker 和 Docker Compose 已安装"
    
    # 检查 Docker 守护进程
    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行，请先启动 Docker"
        exit 1
    fi
    
    log_success "Docker 守护进程运行正常"
    
    # 检查端口占用
    check_port 3000 "用户前端"
    check_port 3001 "管理后台"
    check_port 8080 "后端 API"
    check_port 5432 "PostgreSQL"
    check_port 6379 "Redis"
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2
    
    if lsof -Pi :${port} -sTCP:LISTEN -t >/dev/null 2>&1 || \
       netstat -tuln 2>/dev/null | grep -q ":${port} " || \
       ss -tuln 2>/dev/null | grep -q ":${port} "; then
        log_warning "端口 ${port} (${service}) 已被占用"
        read -p "是否继续? 这可能会导致端口冲突 [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "安装已取消"
            exit 0
        fi
    fi
}

# 获取安装目录
get_install_dir() {
    DEFAULT_INSTALL_DIR="${HOME}/maas-router"
    
    read -p "请输入安装目录 [默认: ${DEFAULT_INSTALL_DIR}]: " INSTALL_DIR
    INSTALL_DIR=${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}
    
    # 展开路径
    INSTALL_DIR="${INSTALL_DIR/#\~/$HOME}"
    
    log_info "安装目录: ${INSTALL_DIR}"
}

# 下载项目代码
download_code() {
    log_info "下载 MaaS-Router 代码..."
    
    # 创建安装目录
    mkdir -p "${INSTALL_DIR}"
    cd "${INSTALL_DIR}"
    
    # 检查是否已存在代码
    if [ -d "${INSTALL_DIR}/.git" ]; then
        log_info "检测到已存在的代码，执行更新..."
        git pull origin main
    else
        # 克隆代码
        # 使用 GitHub 镜像加速（国内）
        GITHUB_MIRROR="https://ghproxy.com/https://github.com"
        REPO_URL="${GITHUB_MIRROR}/your-username/maas-router.git"
        
        log_info "从 ${REPO_URL} 克隆代码..."
        if git clone "${REPO_URL}" . 2>/dev/null; then
            log_success "代码下载成功"
        else
            log_warning "GitHub 镜像下载失败，尝试直接下载..."
            # 备用方案：直接下载压缩包
            curl -L -o maas-router.zip "https://github.com/your-username/maas-router/archive/refs/heads/main.zip"
            unzip -q maas-router.zip
            mv maas-router-main/* .
            mv maas-router-main/.* . 2>/dev/null || true
            rm -rf maas-router-main maas-router.zip
        fi
    fi
    
    log_success "代码准备完成"
}

# 配置环境变量
configure_env() {
    log_info "配置环境变量..."
    
    # 创建 .env 文件
    cat > .env << EOF
# MaaS-Router 环境配置
# 生成时间: $(date)

# 时区
TZ=Asia/Shanghai

# 数据库配置
POSTGRES_USER=maas
POSTGRES_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 16)
POSTGRES_DB=maas_router

# JWT 密钥
JWT_SECRET=$(openssl rand -base64 64)

# 支付配置（可选，如需支付功能请填写）
# Stripe
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=

# 支付宝
ALIPAY_APP_ID=
ALIPAY_PRIVATE_KEY=

# 微信支付
WECHAT_APP_ID=
WECHAT_MCH_ID=
WECHAT_API_KEY=
EOF
    
    log_success "环境配置文件已创建: ${INSTALL_DIR}/.env"
    log_info "请根据需要编辑 .env 文件配置支付等高级功能"
}

# 构建和启动服务
build_and_start() {
    log_info "构建 Docker 镜像..."
    
    # 构建镜像
    ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml build --no-cache
    
    log_success "Docker 镜像构建完成"
    
    log_info "启动服务..."
    
    # 启动服务
    ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml up -d
    
    log_success "服务已启动"
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务就绪..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        echo -n "."
        
        # 检查后端健康状态
        if curl -s http://localhost:8080/health >/dev/null 2>&1; then
            echo ""
            log_success "所有服务已就绪!"
            return 0
        fi
        
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo ""
    log_warning "服务启动可能需要更多时间，请稍后手动检查状态"
    return 1
}

# 显示访问信息
show_access_info() {
    echo ""
    echo "=========================================="
    echo "   MaaS-Router 安装完成!"
    echo "=========================================="
    echo ""
    echo "访问地址:"
    echo "  - 用户前端:    http://localhost:3000"
    echo "  - 管理后台:    http://localhost:3001"
    echo "  - 后端 API:    http://localhost:8080"
    echo "  - API 文档:    http://localhost:8080/swagger"
    echo ""
    echo "数据库连接信息:"
    echo "  - PostgreSQL:  localhost:5432"
    echo "  - Redis:       localhost:6379"
    echo ""
    echo "常用命令:"
    echo "  - 查看日志:    ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml logs -f"
    echo "  - 停止服务:    ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml stop"
    echo "  - 重启服务:    ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml restart"
    echo "  - 更新代码:    cd ${INSTALL_DIR} && git pull && ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml up -d --build"
    echo ""
    echo "配置文件位置:"
    echo "  - 环境变量:    ${INSTALL_DIR}/.env"
    echo ""
    echo "=========================================="
    echo ""
}

# 主函数
main() {
    print_banner
    
    # 检查是否为 root 用户
    if [ "$EUID" -eq 0 ]; then
        log_warning "不建议使用 root 用户运行此脚本"
        read -p "是否继续? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 0
        fi
    fi
    
    # 执行安装步骤
    check_requirements
    get_install_dir
    
    # 确认安装
    echo ""
    read -p "确认开始安装 MaaS-Router? [Y/n]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        log_info "安装已取消"
        exit 0
    fi
    
    download_code
    configure_env
    build_and_start
    wait_for_services
    show_access_info
}

# 处理命令行参数
case "${1:-}" in
    --help|-h)
        echo "MaaS-Router 一键安装脚本"
        echo ""
        echo "使用方法:"
        echo "  $0              交互式安装"
        echo "  $0 --uninstall  卸载 MaaS-Router"
        echo "  $0 --status     查看服务状态"
        echo "  $0 --logs       查看服务日志"
        echo "  $0 --help       显示帮助信息"
        exit 0
        ;;
    --uninstall|-u)
        log_info "卸载 MaaS-Router..."
        if [ -f "${HOME}/maas-router/docker-compose.all-in-one.yml" ]; then
            cd "${HOME}/maas-router"
            ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml down -v
            log_success "服务已停止并移除"
            read -p "是否删除数据卷? 这将删除所有数据 [y/N]: " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                docker volume rm maas-router_postgres_data maas-router_redis_data maas-router_logs 2>/dev/null || true
                log_success "数据卷已删除"
            fi
        fi
        exit 0
        ;;
    --status|-s)
        if [ -f "${HOME}/maas-router/docker-compose.all-in-one.yml" ]; then
            cd "${HOME}/maas-router"
            ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml ps
        else
            log_error "未找到安装目录"
        fi
        exit 0
        ;;
    --logs|-l)
        if [ -f "${HOME}/maas-router/docker-compose.all-in-one.yml" ]; then
            cd "${HOME}/maas-router"
            ${DOCKER_COMPOSE} -f docker-compose.all-in-one.yml logs -f
        else
            log_error "未找到安装目录"
        fi
        exit 0
        ;;
    *)
        main
        ;;
esac
