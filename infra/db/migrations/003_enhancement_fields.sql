-- 003_enhancement_fields.sql
-- P0-1/P0-2/P1-1/P1-2: 增强功能相关数据库迁移
-- 包含：模型映射表、用户分组字段、账号分组权限字段、重试追踪字段

-- ============================================================
-- P0-2: 模型映射表
-- ============================================================
CREATE TABLE IF NOT EXISTS model_mappings (
    id SERIAL PRIMARY KEY,
    account_id VARCHAR(64) DEFAULT NULL,
    source_model VARCHAR(128) NOT NULL,
    target_model VARCHAR(128) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, source_model)
);

-- 按账号索引模型映射（NULL account_id 表示全局映射）
CREATE INDEX idx_model_mappings_account ON model_mappings(account_id);

-- 按源模型名称索引（用于快速查找映射）
CREATE INDEX idx_model_mappings_source ON model_mappings(source_model);

-- ============================================================
-- P1-1: 用户分组字段
-- ============================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS group_name VARCHAR(64) DEFAULT 'default';

-- 按分组名称索引（用于按分组筛选用户）
CREATE INDEX IF NOT EXISTS idx_users_group ON users(group_name);

-- ============================================================
-- P1-1: 账号允许的分组字段
-- ============================================================
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS allowed_groups JSONB DEFAULT '["default"]'::jsonb;

-- ============================================================
-- P0-1: 使用记录中的重试追踪字段
-- ============================================================
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;

ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS retry_attempt INTEGER DEFAULT 0;

-- ============================================================
-- 更新 updated_at 触发器（如果存在）
-- ============================================================
-- 注意：如果已有 updated_at 自动更新触发器，无需额外操作
-- 以下为 model_mappings 表创建触发器
CREATE OR REPLACE FUNCTION update_model_mappings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_model_mappings_updated_at ON model_mappings;
CREATE TRIGGER trg_model_mappings_updated_at
    BEFORE UPDATE ON model_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_model_mappings_updated_at();
