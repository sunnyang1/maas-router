.PHONY: help up down seed dev install clean

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

db-create: ## 创建数据库
	psql -U maas -h localhost -c "CREATE DATABASE maas_router" || true
