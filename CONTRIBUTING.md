# 贡献指南

感谢你对 MaaS-Router 的关注！本文档帮助你了解如何参与项目贡献。

---

## 行为准则

- 尊重每一位贡献者
- 建设性的 Code Review，对事不对人
- 保持沟通透明，有问题及时提出

---

## 我能贡献什么？

| 贡献类型 | 说明 |
|----------|------|
| 🐛 **Bug 修复** | 发现并修复 Bug |
| ✨ **新功能** | 添加新的路由策略、模型供应商接入等 |
| 📝 **文档** | 修正、补充、翻译文档 |
| 🧪 **测试** | 添加单元测试、集成测试 |
| 🎨 **UI/UX** | 改进管理平台界面和交互 |
| ⚡ **性能优化** | 数据库查询优化、缓存策略等 |

---

## 开发流程

### 1. 前期准备

```bash
# Fork 并 Clone 项目
git clone <your-fork-url>
cd maas-router
git remote add upstream <main-repo-url>

# 搭建开发环境（详见 docs/DEVELOPMENT.md）
docker compose up -d postgres redis
make install
make seed
```

### 2. 创建分支

```bash
git checkout main
git pull upstream main
git checkout -b feat/my-feature
```

分支命名规范：`feat/<描述>` / `fix/<描述>` / `docs/<描述>`

### 3. 开发和提交

遵循 [Git 工作流](GIT_WORKFLOW.md) 规范：

```bash
git add backend/app/models/new_model.py
git commit -m "feat(db): add new model table"
```

### 4. 提交前检查

- [ ] 代码通过 lint 检查
- [ ] 已添加/更新测试
- [ ] 已更新相关文档
- [ ] Commit 历史已整理干净

### 5. 创建 Pull Request

```bash
gh pr create \
  --title "feat: add new feature description" \
  --body "## 变更说明

  - 变更点 1
  - 变更点 2

  ## 测试

  - [ ] 单元测试通过
  - [ ] 手动测试通过" \
  --base main
```

---

## 代码规范

### Python（后端）

- 遵循 [PEP 8](https://peps.python.org/pep-0008/)
- 使用 `black` 格式化（行宽 100）
- 使用 `flake8` 检查代码质量
- 类型注解：所有函数参数和返回值应有类型注解

```python
# ✅ 推荐
async def get_user_by_email(email: str, db: AsyncSession) -> User | None:
    result = await db.execute(select(User).where(User.email == email))
    return result.scalar_one_or_none()

# ❌ 不推荐
async def get_user_by_email(email, db):
    result = await db.execute(select(User).where(User.email == email))
    return result.scalar_one_or_none()
```

### TypeScript / React（前端）

- 使用 Prettier 格式化
- 组件使用函数式写法 + Hooks
- Props 必须定义 TypeScript 接口

```tsx
// ✅ 推荐
interface UserCardProps {
  user: User;
  onEdit: (id: string) => void;
}

export const UserCard: React.FC<UserCardProps> = ({ user, onEdit }) => {
  // ...
};
```

### Commit Message

遵循 [Conventional Commits](https://www.conventionalcommits.org/)：

```
<type>(<scope>): <subject>

feat(admin): add model routing rule editor
fix(api): correct billing calculation for tiered pricing
docs: add API authentication guide
```

详细规范见 [Git 工作流](GIT_WORKFLOW.md)。

---

## 文档规范

- 使用 Markdown 格式
- 每个文档有清晰的标题层级
- 代码示例必须可运行
- 新功能必须同步更新文档

---

## 测试指南

```bash
# 后端测试
cd backend
pip install pytest pytest-asyncio
pytest

# 前端测试（待完善）
cd admin-platform
npm test
```

---

## Issue 规范

### Bug Report

```markdown
### 描述
<!-- 简要描述 Bug -->

### 复现步骤
1. 
2. 
3. 

### 期望行为
<!-- 期望发生什么 -->

### 实际行为
<!-- 实际发生什么 -->

### 环境
- OS: 
- Python/Node 版本: 
- 分支/Commit: 
```

### Feature Request

```markdown
### 需求背景
<!-- 为什么需要这个功能 -->

### 功能描述
<!-- 描述期望的功能 -->

### 备选方案
<!-- 是否考虑过替代方案 -->
```

---

## 获取帮助

- 提交 Issue 提问
- 查看 [技术文档](INDEX.md)
- 查看 [故障排查指南](TROUBLESHOOTING.md)
