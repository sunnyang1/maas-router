-- 复杂度分析字段
-- 为 usage_records 表添加复杂度分析相关字段
-- Migration: 002_complexity_fields

ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS complexity_score DOUBLE PRECISION DEFAULT 0;
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS complexity_level VARCHAR(20);
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS routing_tier VARCHAR(20);
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS complexity_model VARCHAR(100);
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS cost_saving_ratio DOUBLE PRECISION DEFAULT 0;
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS quality_risk VARCHAR(20);
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS was_upgraded BOOLEAN DEFAULT FALSE;

-- 索引：按复杂度级别查询
CREATE INDEX IF NOT EXISTS idx_usage_records_complexity_level ON usage_records(complexity_level);

-- 索引：按路由层级和创建时间查询（用于成本分析）
CREATE INDEX IF NOT EXISTS idx_usage_records_routing_tier_created ON usage_records(routing_tier, created_at);

-- 索引：按复杂度级别和路由层级联合查询（用于路由优化分析）
CREATE INDEX IF NOT EXISTS idx_usage_records_complexity_tier ON usage_records(complexity_level, routing_tier);
