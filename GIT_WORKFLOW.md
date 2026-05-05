# 🔀 MaaS-Router 团队 Git 工作流规范

> **适用于**: MaaS-Router 全栈项目  
> **策略**: Trunk-Based Development（主干开发）  
> **更新时间**: 2026-05-05

---

## 📋 目录

1. [核心原则](#1-核心原则)
2. [分支策略](#2-分支策略)
3. [Commit 规范](#3-commit-规范)
4. [日常开发流程](#4-日常开发流程)
5. [Pull Request / Code Review](#5-pull-request--code-review)
6. [冲突解决指南](#6-冲突解决指南)
7. [Git Hooks 与自动化](#7-git-hooks-与自动化)
8. [CI/CD 集成](#8-cicd-集成)
9. [紧急修复流程](#9-紧急修复流程)
10. [常见场景速查表](#10-常见场景速查表)
11. [团队约定与禁忌](#11-团队约定与禁忌)

---

## 1. 核心原则

```
┌─────────────────────────────────────────────────────────────┐
│  🎯 一句话：main 分支始终可部署，所有改动通过短分支 + PR 完成  │
└─────────────────────────────────────────────────────────────┘
```

### 四大铁律

| # | 原则 | 说明 |
|---|------|------|
| 1 | **Commit 原子化** | 每个 commit 只做一件事，可独立 revert |
| 2 | **主干可部署** | `main` 分支任何时候都能安全部署 |
| 3 | **短生命周期分支** | 功能分支存活不超过 2 个工作日 |
| 4 | **PR + Review 必做** | 所有代码变更必须经过 Code Review |

### 为什么选 Trunk-Based？

本项目是 **3-8 人小团队、持续交付型产品**，Trunk-Based 相比 Git Flow 的优势：

| 维度 | Trunk-Based | Git Flow |
|------|-------------|----------|
| 分支复杂度 | ⭐ 简单（main + 短分支） | ⭐⭐⭐ 复杂（main/develop/release/hotfix） |
| 合并冲突 | 少（频繁小步合并） | 多（长时间不合并） |
| 发布频率 | 随时可发布 | 按 release 节奏 |
| 回滚难度 | 低（单 commit revert） | 高（跨多个分支） |
| 适合团队 | 1-20 人 | 大型版本化产品 |

---

## 2. 分支策略

### 2.1 分支模型

```
main ────●────●────●────●────●────●─── (始终可部署，受保护)
          \   /    \   /    \   /
           ●─●      ●─●      ●──      (短生命周期功能分支)
       feat/xxx   fix/xxx   chore/xxx
```

### 2.2 分支命名规范

```
feat/<简短描述>      新功能开发      feat/user-auth, feat/model-routing
fix/<简短描述>        Bug 修复       fix/login-redirect, fix/billing-calc
chore/<简短描述>      基础设施/工具   chore/deps-update, chore/ci-setup
refactor/<简短描述>   代码重构        refactor/api-layer
docs/<简短描述>       文档更新        docs/api-reference
test/<简短描述>       测试相关        test/billing-unit-tests
```

### 2.3 分支生命周期

```
创建 ──→ 开发(1-2天) ──→ rebase main ──→ 推 PR ──→ Review ──→ 合并 ──→ 删除
```

**关键规则**：
- ⚠️ 超过 2 天未合并的分支需要同步 `main` 并推动合并
- ⚠️ 分支合并后 **立即删除远程和本地分支**
- ❌ 禁止直接在 `main` 上 commit
- ❌ 禁止使用 `git push --force` 到共享分支

---

## 3. Commit 规范

### 3.1 Conventional Commits 格式

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

### 3.2 Type 类型

| Type | 用途 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(admin): add model routing rule editor` |
| `fix` | Bug 修复 | `fix(api): correct billing calculation for tiered pricing` |
| `refactor` | 重构（无功能变化） | `refactor(backend): extract auth middleware` |
| `chore` | 构建/工具/依赖 | `chore: update FastAPI to 0.115.0` |
| `docs` | 文档 | `docs: add API authentication guide` |
| `test` | 测试 | `test(billing): add unit tests for credit deduction` |
| `style` | 格式调整 | `style: format with black and prettier` |
| `perf` | 性能优化 | `perf(db): add index on api_keys.user_id` |
| `ci` | CI/CD 变更 | `ci: add backend lint check to pipeline` |

### 3.3 Scope（本项目特定）

```
admin     - 管理平台前端
api       - API Server
backend   - 后端共享模块
billing   - 计费模块
models    - 模型管理
auth      - 认证授权
db        - 数据库
infra     - 基础设施（Docker, CI）
```

### 3.4 正确 vs 错误示例

```bash
# ✅ 正确
git commit -m "feat(admin): add real-time request monitoring chart"
git commit -m "fix(api): handle empty model list in routing"
git commit -m "refactor(backend): unify error response format"

# ❌ 错误
git commit -m "fix bug"                    # 太模糊
git commit -m "WIP"                        # 无意义
git commit -m "feat: add stuff and fix things"  # 一件事一个 commit
git commit -m "fixed the thing finally!!!" # 不规范
```

### 3.5 Atomic Commit 原则

一个好的 commit 满足：
- ✅ 只做一件事（一个逻辑变更）
- ✅ 可以独立 revert 而不破坏其他功能
- ✅ commit message 能完全描述做了什么
- ✅ Review 者能一眼看懂改动范围

---

## 4. 日常开发流程

### 4.1 开始新功能

```bash
# Step 1: 同步最新代码
git checkout main
git pull origin main

# Step 2: 创建功能分支
git checkout -b feat/my-feature

# Step 3: 开发 + 频繁提交
# ... 写代码 ...
git add backend/app/models/new_model.py
git commit -m "feat(db): add ProviderModel table with FK to provider"

git add backend/app/admin_server/models_admin.py
git commit -m "feat(admin): add model CRUD endpoints"
```

### 4.2 清理提交历史（准备 PR 前）

```bash
# Step 1: 拉取最新 main
git fetch origin main

# Step 2: 交互式 rebase，整理 commit
git rebase -i origin/main

# 在编辑器中：
# pick abc1234 feat(db): add ProviderModel table
# squash def5678 WIP: fix typo
# squash ghi9012 fix: update field name again
# reword jkl3456 feat(admin): add model CRUD endpoints
# → 合并为 2 个干净 commit

# Step 3: 推送到远程（⚠️ 仅在个人分支上使用）
git push --force-with-lease origin feat/my-feature
```

### 4.3 创建 Pull Request

```bash
# 使用 GitHub CLI（推荐）
gh pr create \
  --title "feat: add model routing management" \
  --body "## 变更说明
- 新增 ProviderModel 数据表
- 实现模型 CRUD 管理接口
- 添加前端路由规则编辑器

## 测试
- [x] 单元测试通过
- [x] 手动测试：创建/编辑/删除模型正常
- [x] API 文档已更新

## 截图
（附上前端界面截图）" \
  --base main \
  --head feat/model-routing
```

### 4.4 PR 模板

在 `.github/pull_request_template.md` 中：

```markdown
## 📝 变更说明
<!-- 简要描述这个 PR 做了什么 -->

## 🔗 关联 Issue
Closes #

## ✅ 检查清单
- [ ] 代码已本地测试
- [ ] 单元测试通过
- [ ] 无新增 lint 警告
- [ ] API 变更已更新文档
- [ ] 数据库迁移已测试（如有）

## 📸 截图
<!-- 前端变更请附截图 -->

## 🔄 部署注意事项
<!-- 是否需要特殊的部署步骤 -->
```

---

## 5. Pull Request / Code Review

### 5.1 Review 流程

```
开发者提交 PR
     │
     ▼
自动检查: lint → test → build
     │
     ▼ (失败则打回)
至少 1 人 Review
     │
     ├── Approve → 合并到 main
     └── Request Changes → 修改后重新请求 Review
```

### 5.2 Reviewer 检查要点

| 维度 | 关注什么 |
|------|---------|
| **正确性** | 逻辑是否正确？边界情况处理？ |
| **安全性** | SQL 注入？API Key 泄露？权限校验？ |
| **性能** | N+1 查询？不必要的大循环？ |
| **可维护性** | 命名清晰？职责单一？ |
| **Commit 质量** | 历史干净？message 清晰？ |

### 5.3 合并策略

```
✅ 推荐: Squash and Merge
   ─ 将分支上所有 commit 压缩为一个，保持 main 历史干净

✅ 可用: Rebase and Merge
   ─ 保留所有 commit，线性历史

⚠️ 慎用: Merge Commit
   ─ 仅当功能分支确实需要保留独立历史时
```

---

## 6. 冲突解决指南

### 6.1 预防冲突

```bash
# 每日开始工作前
git fetch origin main
git rebase origin/main   # 在自己的分支上 rebase

# 如果 rebase 过程中有冲突
# 1. 解决冲突文件
# 2. git add <resolved-file>
# 3. git rebase --continue
# 4. 如果想放弃: git rebase --abort
```

### 6.2 解决冲突的标准流程

```bash
# Step 1: 确保当前工作区干净
git status

# Step 2: Rebase 到最新 main
git fetch origin
git rebase origin/main

# Step 3: 如有冲突，逐个解决
# 冲突标记长这样：
# <<<<<<< HEAD          ← main 的版本
#   code from main
# =======
#   code from your branch
# >>>>>>> feat/xxx      ← 你的版本

# Step 4: 解决后继续
git add <resolved-file>
git rebase --continue

# Step 5: 推送到远程
git push --force-with-lease origin feat/xxx
```

---

## 7. Git Hooks 与自动化

### 7.1 Pre-commit Hooks（推荐设置）

安装 pre-commit 并在 `.pre-commit-config.yaml` 中配置：

```yaml
repos:
  # Python 后端
  - repo: https://github.com/psf/black
    rev: 24.4.0
    hooks:
      - id: black
        args: [--line-length=100]
        files: ^backend/

  - repo: https://github.com/PyCQA/flake8
    rev: 7.0.0
    hooks:
      - id: flake8
        files: ^backend/

  # 前端
  - repo: https://github.com/pre-commit/mirrors-prettier
    rev: v4.0.0-alpha.8
    hooks:
      - id: prettier
        files: ^admin-platform/

  # 通用
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-json
      - id: detect-private-key
```

### 7.2 Commit Message 校验（推荐）

```bash
# 安装 commitlint（Node 项目）
cd admin-platform && npm install --save-dev @commitlint/cli @commitlint/config-conventional
```

`commitlint.config.js`:
```js
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', [
      'feat', 'fix', 'refactor', 'chore', 'docs',
      'test', 'style', 'perf', 'ci', 'revert'
    ]],
    'scope-enum': [2, 'always', [
      'admin', 'api', 'backend', 'billing', 'models',
      'auth', 'db', 'infra', 'deps'
    ]]
  }
};
```

---

## 8. CI/CD 集成

### 8.1 推荐的 GitHub Actions 流程

```yaml
# .github/workflows/ci.yml
name: CI Pipeline

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  lint-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install black flake8
      - run: black --check backend/
      - run: flake8 backend/

  lint-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: cd admin-platform && npm ci && npm run lint

  test-backend:
    runs-on: ubuntu-latest
    needs: lint-backend
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: test_db
        ports: ['5432:5432']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install -r backend/requirements.txt
      - run: cd backend && pytest

  build:
    runs-on: ubuntu-latest
    needs: [test-backend]
    steps:
      - uses: actions/checkout@v4
      - run: docker compose build
```

### 8.2 分支保护规则（GitHub Settings）

在仓库 Settings → Branches → Add rule 中设置：

| 规则 | 值 |
|------|---|
| Branch name pattern | `main` |
| Require a pull request before merging | ✅ |
| Required approvals | 1 |
| Dismiss stale reviews | ✅ |
| Require status checks to pass | ✅ |
| Require branches to be up to date | ✅ |
| Include administrators | ✅ |

---

## 9. 紧急修复流程

当生产环境出现需要立即修复的 Bug：

```bash
# Step 1: 从 main 创建 hotfix 分支
git checkout main
git pull origin main
git checkout -b fix/critical-billing-error

# Step 2: 修复 + commit
git add backend/app/api_server/billing.py
git commit -m "fix(billing): prevent double credit deduction on retry"

# Step 3: 创建 PR（加 urgent 标签）
gh pr create \
  --title "🔴 URGENT: fix(billing): prevent double credit deduction" \
  --label "urgent" \
  --base main

# Step 4: Review 通过后，Squash Merge
# Step 5: 立即部署
make deploy
```

---

## 10. 常见场景速查表

### 场景一：我搞砸了，需要撤销

```bash
# 撤销最后一次 commit（保留改动）
git reset --soft HEAD~1

# 撤销最后一次 commit（丢弃改动）⚠️
git reset --hard HEAD~1

# 撤销已推送的 commit（通过新建 revert commit）
git revert HEAD
git push origin main

# 临时保存当前改动
git stash
git stash pop   # 恢复

# 丢弃工作区所有改动 ⚠️
git checkout -- .
```

### 场景二：分支管理

```bash
# 查看所有分支及状态
git branch -a -v

# 删除已合并的本地分支
git branch --merged | grep -v "main" | xargs git branch -d

# 删除远程分支
git push origin --delete feat/old-feature

# 重命名分支
git branch -m old-name new-name
git push origin --delete old-name
git push origin new-name
```

### 场景三：查找问题

```bash
# 查看某个文件的修改历史
git log -p -- backend/app/core/security.py

# 查看谁改了哪行（blame）
git blame backend/app/core/security.py

# 二分查找引入 bug 的 commit
git bisect start
git bisect bad HEAD
git bisect good v1.0.0
# ... git 会自动二分查找 ...

# 查看某次 commit 的详细内容
git show abc1234

# 搜索 commit message
git log --grep="billing"
```

### 场景四：并行开发多个功能（Worktree）

```bash
# 创建独立工作目录，同时开发两个功能
git worktree add ../maas-feat-auth feat/user-auth
git worktree add ../maas-fix-billing fix/billing

# 查看所有 worktree
git worktree list

# 完成后删除
git worktree remove ../maas-feat-auth
```

---

## 11. 团队约定与禁忌

### ✅ 必须遵守

| # | 约定 |
|---|------|
| 1 | 每天开始工作前 `git fetch origin && git rebase origin/main` |
| 2 | 分支存活不超过 2 天（超期需同步 main 并推动合并） |
| 3 | PR 提交前 rebase 整理 commit 历史 |
| 4 | Code Review 在 4 小时内响应（工作时间） |
| 5 | 合并后立即删除远程分支 |
| 6 | 使用 `--force-with-lease`，绝不使用 `--force` |

### ❌ 严格禁止

| # | 禁忌 | 后果 |
|---|------|------|
| 1 | **直接 push 到 main** | 绕过 Review，可能引入问题 |
| 2 | **force push 到 main 或共享分支** | 覆盖他人代码 |
| 3 | **提交密钥/密码/Token** | 安全漏洞，需轮换所有密钥 |
| 4 | **合并失败的 CI 的 PR** | 破坏主干稳定性 |
| 5 | **在 PR 中包含无关改动** | 增大 Review 难度，引入风险 |
| 6 | **提交 node_modules 或其他生成文件** | 仓库膨胀，无意义变更 |

### ⚠️ 需要团队讨论

| # | 场景 | 建议 |
|---|------|------|
| 1 | 大规模重构 | 使用 Feature Flag，分批提交 PR |
| 2 | 数据库迁移 | 必须可回滚，先在 staging 验证 |
| 3 | 第三方依赖大版本升级 | 单独 PR，充分测试 |
| 4 | API Breaking Change | 提前通知，做好版本管理 |

---

## 📎 附录

### A. 快速配置新成员

```bash
# 新成员加入团队时执行
git clone <repo-url>
cd maas-router

# 配置用户信息
git config user.name "Your Name"
git config user.email "your@email.com"

# 设置默认分支
git config init.defaultBranch main

# 安装 pre-commit hooks
pip install pre-commit
pre-commit install

# 配置 pull 策略为 rebase（避免无意义的 merge commit）
git config pull.rebase true
git config rebase.autoStash true
```

### B. 有用别名

```bash
# 添加到 ~/.gitconfig
[alias]
  # 漂亮的 log
  lg = log --graph --pretty=format:'%Cred%h%Creset -%C(yellow)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' --abbrev-commit
  # 查看当前分支状态
  st = status -sb
  # 查看未推送的 commits
  unpushed = log --branches --not --remotes
  # 删除已合并分支
  cleanup = !git branch --merged | grep -v 'main' | xargs git branch -d
  # 修改上次 commit message
  amend = commit --amend
  # 撤销最后一次 commit
  undo = reset --soft HEAD~1
```

---

> 💡 **记住**: 好的 Git 工作流不是束缚，而是让团队合作更顺畅的轨道。规范越早建立，后期越少痛苦。
