# MaaS-Router 文档自动化方案

> 基于现有 FastAPI + PostgreSQL + Redis 架构，实现六大类文档的按需生成与定时调度。

---

## 一、架构总览

```
┌─────────────────────────────────────────────────────────┐
│                     Admin Platform (React)                │
│               「文档中心」页面 → 一键生成 / 下载            │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP REST
┌─────────────────────▼───────────────────────────────────┐
│              Admin Server (FastAPI :8005)                │
│   /api/admin/v1/documents/                               │
│   ├── POST /generate/billing      计费报表 (PDF/Excel)    │
│   ├── POST /generate/user-report  用户报告 (PDF/Word)     │
│   ├── POST /generate/ops-daily    运维日报 (PDF)         │
│   ├── POST /generate/audit        审计报告 (Word)        │
│   ├── POST /generate/data-export  数据导出 (Excel/CSV)   │
│   ├── GET  /list                  文档列表               │
│   └── GET  /download/{filename}   下载文档               │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│              DocumentService (Facade)                     │
│   ┌──────────┬──────────┬──────────┬──────────┐         │
│   │PDFEngine │ExcelEng. │WordEngine│PPTXEngine│         │
│   │WeasyPrint│openpyxl  │python-   │python-   │         │
│   │          │          │docx      │pptx      │         │
│   └──────────┴──────────┴──────────┴──────────┘         │
│                                                          │
│   DataAdapter: DB queries → template context             │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                   Data Sources                           │
│   PostgreSQL: users, transactions, request_logs,         │
│               audit_logs, api_keys, models               │
│   Redis: real-time metrics, rate limits                  │
└─────────────────────────────────────────────────────────┘
```

## 二、技术选型与理由

| 格式 | 引擎 | 选型理由 |
|------|------|----------|
| **PDF** | WeasyPrint | HTML+CSS→PDF，支持复杂排版、品牌化样式，比 ReportLab 开发效率高 5x |
| **Excel** | openpyxl | 原生 .xlsx，支持多 Sheet、图表、公式、条件格式，pivot-ready |
| **Word** | python-docx | 完整样式系统、目录、页眉页脚，适合审计/合规报告 |
| **PPTX** | python-pptx | 数据驱动幻灯片，适合季度汇报、投资人演示 |
| **CSV** | Python csv | 轻量数据导出，通用性强 |

## 三、文件结构

```
backend/app/
├── services/
│   └── document_service.py    ← 核心引擎（PDF/Excel/Word/PPTX + 统一门面）
├── templates/
│   └── documents/
│       ├── billing_report.html  ← 计费报表模板
│       ├── user_report.html     ← 用户报告模板
│       └── ops_daily.html       ← 运维日报模板
├── admin_server/
│   └── documents.py           ← REST API 端点
└── output/                    ← 生成的文档存放目录
```

## 四、六大文档类型矩阵

| # | 文档类型 | 格式 | 数据源 | 场景 |
|---|---------|------|--------|------|
| 1 | **计费报表** | PDF / Excel | transactions, request_logs | 月度对账、收入分析 |
| 2 | **用户使用报告** | PDF / Word | users, request_logs, transactions | 客户运营、升级引导 |
| 3 | **运维日报** | PDF | request_logs, 服务健康检查 | 每日站会、故障回顾 |
| 4 | **审计报告** | Word | audit_logs | 合规审查、SOC 2 |
| 5 | **模型性能报告** | PPTX | request_logs, routing decisions | 季度汇报、技术评审 |
| 6 | **数据导出** | Excel / CSV | 任意表 | 数据分析、BI 对接 |

## 五、API 使用示例

### 5.1 生成计费报表 (PDF)
```bash
curl -X POST http://localhost:8005/api/admin/v1/documents/generate/billing \
  -H "Content-Type: application/json" \
  -d '{"doc_type": "billing_report", "format": "pdf", "period_days": 30}'
```

### 5.2 生成计费报表 (Excel)
```bash
curl -X POST http://localhost:8005/api/admin/v1/documents/generate/billing \
  -H "Content-Type: application/json" \
  -d '{"doc_type": "billing_report", "format": "xlsx", "period_days": 30}'
```

### 5.3 生成用户使用报告
```bash
curl -X POST http://localhost:8005/api/admin/v1/documents/generate/user-report \
  -H "Content-Type: application/json" \
  -d '{"doc_type": "user_usage_report", "format": "pdf", "period_days": 30}'
```

### 5.4 生成数据导出
```bash
curl -X POST http://localhost:8005/api/admin/v1/documents/generate/data-export \
  -H "Content-Type: application/json" \
  -d '{"doc_type": "data_export", "format": "xlsx", "period_days": 90}'
```

### 5.5 下载文档
```bash
curl -O http://localhost:8005/api/admin/v1/documents/download/billing_report_20260505_120000.pdf
```

## 六、安装与启动

```bash
# 1. 安装文档自动化依赖
make docs-install

# 2. 检查引擎状态
make docs-status
# → {"pdf": true, "excel": true, "word": true, "pptx": true}

# 3. 启动 Admin Server
make dev-admin

# 4. 访问 API 文档
open http://localhost:8005/docs
```

## 七、扩展建议

### 7.1 定时调度（短期）
使用 FastAPI BackgroundTasks + APScheduler 实现每日自动生成运维日报：

```python
# backend/app/scheduler.py (待实现)
from apscheduler.schedulers.asyncio import AsyncIOScheduler

scheduler = AsyncIOScheduler()
scheduler.add_job(generate_daily_ops_report, 'cron', hour=8, minute=0)
```

### 7.2 消息推送（中期）
生成完成后通过企业微信/飞书/Slack 推送通知和下载链接。

### 7.3 前端集成（中期）
在 admin-platform 中增加「文档中心」页面：
- 选择文档类型 + 时间范围 + 格式 → 一键生成
- 文档列表 + 预览 + 下载
- 定时任务管理（启用/禁用/修改频率）

### 7.4 自定义模板（长期）
- 支持用户上传自定义 Jinja2/HTML 模板
- 模板变量编辑器 + 实时预览
- 模板版本管理
