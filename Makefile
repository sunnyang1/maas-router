.PHONY: help up down seed dev install clean db-migrate db-upgrade db-downgrade \
        deploy-staging deploy-prod rollback health-check backup db-backup prod-up prod-down prod-logs

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

up: ## 启动所有服务 (Docker)
	docker compose up -d

down: ## 停止所有服务
	docker compose down

logs: ## 查看日志
	docker compose logs -f

seed: ## 初始化数据库种子数据
	cd backend && python -m app.scripts.seed

install: ## 安装所有依赖
	cd backend && pip install -r requirements.txt
	cd admin-platform && npm install

dev-api: ## 启动 API Server (本地)
	cd backend && uvicorn app.main:app --host 0.0.0.0 --port 8001 --reload

dev-admin: ## 启动 Admin Server (本地)
	cd backend && uvicorn app.admin_main:app --host 0.0.0.0 --port 8005 --reload

dev-frontend: ## 启动管理平台前端
	cd admin-platform && npm run dev

clean: ## 清理
	rm -rf admin-platform/node_modules admin-platform/dist
	find backend -name "__pycache__" -exec rm -rf {} +
	rm -rf backend/app/output/*

db-create: ## 创建数据库
	psql -U maas -h localhost -c "CREATE DATABASE maas_router" || true

# ── Database Migrations (Alembic) ──────────────────────

db-migrate: ## 生成数据库迁移 (alembic revision --autogenerate)
	cd backend && alembic revision --autogenerate -m "auto"

db-upgrade: ## 执行所有待处理的迁移 (alembic upgrade head)
	cd backend && alembic upgrade head

db-downgrade: ## 回滚上一个迁移 (alembic downgrade -1)
	cd backend && alembic downgrade -1

db-history: ## 查看迁移历史
	cd backend && alembic history

db-current: ## 查看当前迁移版本
	cd backend && alembic current

# ── Documentation ──────────────────────────────────────
docs-install-site: ## 安装文档站点依赖 (MkDocs + Material)
	pip install mkdocs mkdocs-material

docs-serve: ## 启动文档站点本地预览 (http://localhost:8000)
	mkdocs serve

docs-build: ## 构建文档站点静态文件
	mkdocs build --strict

docs-lint: ## 检查文档规范（需要 markdownlint-cli）
	@which markdownlint >/dev/null 2>&1 || (echo "安装 markdownlint: npm install -g markdownlint-cli" && exit 1)
	markdownlint "docs/**/*.md" "*.md" --config .markdownlint.json

docs-deploy: ## 部署文档站点到 GitHub Pages
	mkdocs gh-deploy --force

# ── Document Automation ─────────────────────────────────
docs-install: ## 安装文档自动化依赖
	cd backend && pip install weasyprint openpyxl python-docx python-pptx jinja2

docs-status: ## 检查文档引擎可用状态
	cd backend && python -c "from app.services.document_service import DocumentService; d = DocumentService(); print(d.get_available_engines())"

docs-list: ## 列出已生成的文档
	ls -lh backend/app/output/

docs-clean: ## 清理已生成的文档
	rm -rf backend/app/output/*

# ── Deployment ──────────────────────────────────────────

deploy-staging: ## 部署到 Staging 环境
	@bash scripts/deploy.sh staging

deploy-prod: ## 部署到 Production 环境（需要确认）
	@bash scripts/deploy.sh production

rollback: ## 回滚到上一个版本
	@bash scripts/rollback.sh

health-check: ## 健康检查
	@echo "Checking API Server..."
	@curl -sf http://localhost:8001/health && echo " ✅" || echo " ❌"
	@echo "Checking Admin Server..."
	@curl -sf http://localhost:8005/health && echo " ✅" || echo " ❌"

backup: db-backup ## 备份数据库（别名）

db-backup: ## 备份 PostgreSQL 数据库
	@mkdir -p data/backups
	docker compose exec -T postgres pg_dump -U maas maas_router | gzip > data/backups/maas_backup_$$(date +%Y%m%d_%H%M%S).sql.gz
	@echo "✅ 数据库已备份到 data/backups/"

# ── Production Docker Compose ──────────────────────────

prod-up: ## 启动生产环境服务
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

prod-down: ## 停止生产环境服务
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-logs: ## 查看生产环境日志
	docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

prod-ps: ## 查看生产环境服务状态
	docker compose -f docker-compose.yml -f docker-compose.prod.yml ps
