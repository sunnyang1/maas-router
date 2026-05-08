-- MaaS-Router 数据库优化脚本
-- 包含索引创建、分区表配置、性能调优等
-- 执行前请确保已备份数据库

-- ============================================
-- 1. 索引优化
-- ============================================

-- 用户表索引优化
-- 邮箱唯一索引（已存在，用于验证）
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 状态索引（用于按状态筛选用户）
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- 角色索引（用于按角色筛选用户）
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 邀请码唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_invite_code ON users(invite_code) WHERE invite_code IS NOT NULL;

-- 邀请人ID索引
CREATE INDEX IF NOT EXISTS idx_users_invited_by ON users(invited_by) WHERE invited_by IS NOT NULL;

-- 复合索引：状态和最后活跃时间（用于查询活跃用户）
CREATE INDEX IF NOT EXISTS idx_users_status_last_active ON users(status, last_active_at);

-- 复合索引：角色和状态（用于管理员查询）
CREATE INDEX IF NOT EXISTS idx_users_role_status ON users(role, status);

-- 创建时间索引（用于排序）
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- ============================================
-- 2. 账号表索引优化
-- ============================================

-- 平台索引
CREATE INDEX IF NOT EXISTS idx_accounts_platform ON accounts(platform);

-- 状态索引
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);

-- 账号类型索引
CREATE INDEX IF NOT EXISTS idx_accounts_account_type ON accounts(account_type);

-- 复合索引：平台和状态（最常用的查询组合）
CREATE INDEX IF NOT EXISTS idx_accounts_platform_status ON accounts(platform, status);

-- 复合索引：状态和最后使用时间（用于调度算法）
CREATE INDEX IF NOT EXISTS idx_accounts_status_last_used ON accounts(status, last_used_at);

-- 最后错误时间索引（用于错误监控）
CREATE INDEX IF NOT EXISTS idx_accounts_last_error_at ON accounts(last_error_at) WHERE last_error_at IS NOT NULL;

-- 创建时间索引
CREATE INDEX IF NOT EXISTS idx_accounts_created_at ON accounts(created_at DESC);

-- ============================================
-- 3. API Key 表索引优化
-- ============================================

-- Key Hash 唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);

-- 用户ID索引
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);

-- 状态索引
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

-- 复合索引：用户ID和状态
CREATE INDEX IF NOT EXISTS idx_api_keys_user_status ON api_keys(user_id, status);

-- 过期时间索引（用于清理过期 Key）
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- 最后使用时间索引
CREATE INDEX IF NOT EXISTS idx_api_keys_last_used_at ON api_keys(last_used_at);

-- 创建时间索引
CREATE INDEX IF NOT EXISTS idx_api_keys_created_at ON api_keys(created_at DESC);

-- ============================================
-- 4. 使用记录表索引优化
-- ============================================

-- 请求ID唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_request_id ON usage_records(request_id);

-- 用户ID索引
CREATE INDEX IF NOT EXISTS idx_usage_records_user_id ON usage_records(user_id);

-- 创建时间索引
CREATE INDEX IF NOT EXISTS idx_usage_records_created_at ON usage_records(created_at DESC);

-- 复合索引：用户ID和创建时间（最常用的查询组合）
CREATE INDEX IF NOT EXISTS idx_usage_records_user_created ON usage_records(user_id, created_at DESC);

-- API Key ID索引
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id ON usage_records(api_key_id) WHERE api_key_id IS NOT NULL;

-- 账号ID索引
CREATE INDEX IF NOT EXISTS idx_usage_records_account_id ON usage_records(account_id) WHERE account_id IS NOT NULL;

-- 分组ID索引
CREATE INDEX IF NOT EXISTS idx_usage_records_group_id ON usage_records(group_id) WHERE group_id IS NOT NULL;

-- 平台索引
CREATE INDEX IF NOT EXISTS idx_usage_records_platform ON usage_records(platform);

-- 状态索引
CREATE INDEX IF NOT EXISTS idx_usage_records_status ON usage_records(status);

-- 复合索引：平台和创建时间（用于平台统计）
CREATE INDEX IF NOT EXISTS idx_usage_records_platform_created ON usage_records(platform, created_at);

-- 模型索引
CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);

-- ============================================
-- 5. 分区表配置（使用记录表）
-- ============================================

-- 创建按月的分区表（如果数据库支持）
-- 注意：PostgreSQL 10+ 支持声明式分区

-- 检查表是否已分区
DO $$
BEGIN
    -- 如果表未分区，创建分区表结构
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE tablename = 'usage_records' 
        AND schemaname = 'public'
    ) THEN
        -- 创建分区表（首次部署时使用）
        CREATE TABLE usage_records (
            id BIGSERIAL,
            request_id VARCHAR(64) NOT NULL,
            user_id BIGINT NOT NULL,
            api_key_id BIGINT,
            account_id BIGINT,
            group_id BIGINT,
            model VARCHAR(100) NOT NULL,
            platform VARCHAR(50) NOT NULL,
            prompt_tokens INTEGER DEFAULT 0,
            completion_tokens INTEGER DEFAULT 0,
            total_tokens INTEGER DEFAULT 0,
            latency_ms INTEGER,
            first_token_ms INTEGER,
            cost DECIMAL(18,6) DEFAULT 0,
            status VARCHAR(20) NOT NULL,
            error_message TEXT,
            client_ip VARCHAR(45),
            user_agent VARCHAR(500),
            created_at TIMESTAMP NOT NULL,
            PRIMARY KEY (id, created_at)
        ) PARTITION BY RANGE (created_at);
    END IF;
END $$;

-- 创建分区（按月）
-- 创建当前月份和前后各 6 个月的分区
DO $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
    start_str TEXT;
    end_str TEXT;
BEGIN
    FOR i IN -6..6 LOOP
        start_date := DATE_TRUNC('month', CURRENT_DATE + (i || ' months')::INTERVAL);
        end_date := start_date + INTERVAL '1 month';
        partition_name := 'usage_records_' || TO_CHAR(start_date, 'YYYY_MM');
        start_str := TO_CHAR(start_date, 'YYYY-MM-DD');
        end_str := TO_CHAR(end_date, 'YYYY-MM-DD');
        
        -- 检查分区是否存在
        IF NOT EXISTS (
            SELECT 1 FROM pg_tables 
            WHERE tablename = partition_name 
            AND schemaname = 'public'
        ) THEN
            EXECUTE format(
                'CREATE TABLE IF NOT EXISTS %I PARTITION OF usage_records 
                 FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_str, end_str
            );
        END IF;
    END LOOP;
END $$;

-- ============================================
-- 6. 数据库参数优化
-- ============================================

-- 共享缓冲区大小（根据服务器内存调整，建议设置为内存的 25%）
-- ALTER SYSTEM SET shared_buffers = '1GB';

-- 有效缓存大小（设置为内存的 50-75%）
-- ALTER SYSTEM SET effective_cache_size = '3GB';

-- 工作内存（用于排序和哈希操作）
-- ALTER SYSTEM SET work_mem = '16MB';

-- 维护工作内存（用于 VACUUM、CREATE INDEX 等）
-- ALTER SYSTEM SET maintenance_work_mem = '256MB';

-- 并发连接数
-- ALTER SYSTEM SET max_connections = 200;

-- WAL 缓冲区
-- ALTER SYSTEM SET wal_buffers = '16MB';

-- 检查点段大小
-- ALTER SYSTEM SET checkpoint_completion_target = 0.9;

-- 随机页面成本（SSD 设置为 1.1，HDD 保持默认 4）
-- ALTER SYSTEM SET random_page_cost = 1.1;

-- 并行工作进程
-- ALTER SYSTEM SET max_parallel_workers_per_gather = 4;
-- ALTER SYSTEM SET max_parallel_workers = 8;

-- 应用配置更改
-- SELECT pg_reload_conf();

-- ============================================
-- 7. 表维护操作
-- ============================================

-- 更新表统计信息
ANALYZE users;
ANALYZE accounts;
ANALYZE api_keys;
ANALYZE usage_records;
ANALYZE groups;
ANALYZE account_groups;
ANALYZE router_rules;
ANALYZE payment_orders;
ANALYZE announcements;

-- 清理表（回收空间）
VACUUM ANALYZE users;
VACUUM ANALYZE accounts;
VACUUM ANALYZE api_keys;
VACUUM ANALYZE usage_records;

-- ============================================
-- 8. 监控视图创建
-- ============================================

-- 表大小监控视图
CREATE OR REPLACE VIEW v_table_sizes AS
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS total_size,
    pg_total_relation_size(schemaname || '.' || tablename) AS total_size_bytes,
    pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size,
    pg_size_pretty(pg_indexes_size(schemaname || '.' || tablename)) AS indexes_size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC;

-- 索引使用情况监控视图
CREATE OR REPLACE VIEW v_index_usage AS
SELECT
    schemaname,
    tablename,
    indexrelname AS index_name,
    idx_scan AS index_scans,
    idx_tup_read AS tuples_read,
    idx_tup_fetch AS tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- 慢查询监控视图
CREATE OR REPLACE VIEW v_slow_queries AS
SELECT
    query,
    calls,
    total_time,
    mean_time,
    max_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_stat_statements%'
ORDER BY mean_time DESC
LIMIT 50;

-- 连接数监控视图
CREATE OR REPLACE VIEW v_connection_stats AS
SELECT
    datname AS database,
    count(*) AS total_connections,
    count(*) FILTER (WHERE state = 'active') AS active_connections,
    count(*) FILTER (WHERE state = 'idle') AS idle_connections,
    count(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_transaction
FROM pg_stat_activity
WHERE datname IS NOT NULL
GROUP BY datname;

-- ============================================
-- 9. 定期维护任务建议
-- ============================================

-- 建议添加到 cron 或 pg_cron 的维护任务：

-- 1. 每小时更新统计信息
-- SELECT cron.schedule('0 * * * *', 'ANALYZE usage_records;');

-- 2. 每天凌晨清理使用记录表
-- SELECT cron.schedule('0 3 * * *', 'VACUUM ANALYZE usage_records;');

-- 3. 每周重建索引（可选，用于碎片整理）
-- SELECT cron.schedule('0 4 * * 0', 'REINDEX TABLE CONCURRENTLY usage_records;');

-- 4. 每月创建新的使用记录分区
-- SELECT cron.schedule('0 2 1 * *', $$
--     DO $$
--     DECLARE
--         start_date DATE := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
--         end_date DATE := start_date + INTERVAL '1 month';
--         partition_name TEXT := 'usage_records_' || TO_CHAR(start_date, 'YYYY_MM');
--     BEGIN
--         EXECUTE format(
--             'CREATE TABLE IF NOT EXISTS %I PARTITION OF usage_records FOR VALUES FROM (%L) TO (%L)',
--             partition_name, start_date, end_date
--         );
--     END $$;
-- $$);

-- ============================================
-- 10. 性能监控查询示例
-- ============================================

-- 查看表大小
-- SELECT * FROM v_table_sizes;

-- 查看索引使用情况
-- SELECT * FROM v_index_usage WHERE index_scans = 0;

-- 查看当前连接数
-- SELECT * FROM v_connection_stats;

-- 查看缓存命中率
-- SELECT
--     sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) AS cache_hit_ratio
-- FROM pg_statio_user_tables;

-- 查看锁等待
-- SELECT
--     blocked_locks.pid AS blocked_pid,
--     blocked_activity.usename AS blocked_user,
--     blocking_locks.pid AS blocking_pid,
--     blocking_activity.usename AS blocking_user,
--     blocked_activity.query AS blocked_statement,
--     blocking_activity.query AS blocking_statement
-- FROM pg_catalog.pg_locks blocked_locks
-- JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
-- JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
--     AND blocking_locks.relation = blocked_locks.relation
--     AND blocking_locks.pid != blocked_locks.pid
-- JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
-- WHERE NOT blocked_locks.granted;

-- 完成提示
SELECT '数据库优化脚本执行完成' AS message;
SELECT '请根据实际服务器配置调整数据库参数' AS note;
SELECT '建议定期运行 ANALYZE 和 VACUUM 维护数据库' AS recommendation;
