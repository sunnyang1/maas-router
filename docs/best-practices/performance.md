# MaaS-Router 性能优化最佳实践

本文档介绍 MaaS-Router 项目的性能优化策略，包括数据库优化、Redis 缓存、连接池配置、负载均衡和监控指标。

## 目录

- [数据库优化](#数据库优化)
- [Redis 缓存策略](#redis-缓存策略)
- [连接池配置](#连接池配置)
- [负载均衡](#负载均衡)
- [监控指标](#监控指标)

## 数据库优化

### 1. PostgreSQL 优化

#### 连接池配置

```yaml
# config.yaml
database:
  host: "localhost"
  port: 5432
  user: "maas_user"
  password: "password"
  database: "maas_router"
  sslmode: "require"
  
  # 连接池配置
  max_open_conns: 100        # 最大打开连接数
  max_idle_conns: 20         # 最大空闲连接数
  conn_max_lifetime: "1h"    # 连接最大生命周期
  conn_max_idle_time: "30m"  # 空闲连接最大存活时间
```

#### PostgreSQL 参数调优

```sql
-- postgresql.conf 优化配置

-- 内存配置（根据服务器内存调整）
shared_buffers = 4GB                    -- 推荐设置为内存的 25%
effective_cache_size = 12GB             -- 推荐设置为内存的 75%
work_mem = 256MB                        -- 每个查询操作的工作内存
maintenance_work_mem = 512MB            -- 维护操作的工作内存

-- WAL 配置
wal_buffers = 16MB
max_wal_size = 4GB
min_wal_size = 1GB
wal_compression = on

-- 并发配置
max_connections = 500
max_worker_processes = 8
max_parallel_workers_per_gather = 4
max_parallel_workers = 8

-- 查询优化
random_page_cost = 1.1                  -- SSD 设置为 1.1，HDD 为 4
effective_io_concurrency = 200          -- SSD 设置为 200

-- 日志配置
log_min_duration_statement = 1000       -- 记录执行超过 1s 的查询
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on
```

#### 索引优化

```sql
-- 用户表索引
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
CREATE INDEX CONCURRENTLY idx_users_api_key ON users(api_key);
CREATE INDEX CONCURRENTLY idx_users_created_at ON users(created_at);

-- 请求日志表索引（分区表）
CREATE INDEX CONCURRENTLY idx_request_logs_user_id ON request_logs(user_id);
CREATE INDEX CONCURRENTLY idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX CONCURRENTLY idx_request_logs_provider ON request_logs(provider_id);
CREATE INDEX CONCURRENTLY idx_request_logs_status ON request_logs(status);

-- 复合索引
CREATE INDEX CONCURRENTLY idx_request_logs_user_created 
ON request_logs(user_id, created_at);

-- 模型提供商索引
CREATE INDEX CONCURRENTLY idx_model_providers_status ON model_providers(status, priority);

-- 使用部分索引优化特定查询
CREATE INDEX CONCURRENTLY idx_request_logs_failed 
ON request_logs(user_id, created_at) 
WHERE status >= 500;
```

#### 查询优化

```sql
-- 使用 EXPLAIN ANALYZE 分析慢查询
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT u.id, u.email, COUNT(r.id) as request_count
FROM users u
LEFT JOIN request_logs r ON u.id = r.user_id
WHERE u.created_at > '2024-01-01'
GROUP BY u.id, u.email
HAVING COUNT(r.id) > 100;

-- 优化后的查询
WITH recent_users AS (
    SELECT id, email
    FROM users
    WHERE created_at > '2024-01-01'
),
user_requests AS (
    SELECT user_id, COUNT(*) as cnt
    FROM request_logs
    WHERE created_at > '2024-01-01'
    GROUP BY user_id
    HAVING COUNT(*) > 100
)
SELECT ru.id, ru.email, ur.cnt as request_count
FROM recent_users ru
JOIN user_requests ur ON ru.id = ur.user_id;
```

#### 分区表策略

```sql
-- 按时间分区（适用于日志表）
CREATE TABLE request_logs (
    id BIGSERIAL,
    user_id BIGINT NOT NULL,
    provider_id INT NOT NULL,
    request_data JSONB,
    response_data JSONB,
    status INT,
    latency_ms INT,
    cost DECIMAL(10,6),
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建分区
CREATE TABLE request_logs_2024_01 PARTITION OF request_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE request_logs_2024_02 PARTITION OF request_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... 更多分区

-- 自动创建分区的函数
CREATE OR REPLACE FUNCTION create_request_logs_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    partition_date := DATE_TRUNC('month', NOW() + INTERVAL '1 month');
    partition_name := 'request_logs_' || TO_CHAR(partition_date, 'YYYY_MM');
    start_date := partition_date;
    end_date := partition_date + INTERVAL '1 month';
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF request_logs FOR VALUES FROM (%L) TO (%L)',
                   partition_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;
```

### 2. 数据库监控

```sql
-- 查看活跃连接
SELECT 
    datname,
    usename,
    application_name,
    client_addr,
    state,
    COUNT(*)
FROM pg_stat_activity
WHERE state IS NOT NULL
GROUP BY datname, usename, application_name, client_addr, state;

-- 查看慢查询
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- 查看表统计信息
SELECT 
    schemaname,
    tablename,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;

-- 查看索引使用情况
SELECT 
    schemaname,
    tablename,
    indexrelname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;
```

## Redis 缓存策略

### 1. 缓存架构设计

```
┌─────────────────┐
│   API Gateway   │
└────────┬────────┘
         │
    ┌────┴────┐
    │  Cache  │  ← L1: 本地缓存 (go-cache)
    └────┬────┘
         │ Cache Miss
    ┌────┴────┐
    │  Redis  │  ← L2: 分布式缓存
    └────┬────┘
         │ Cache Miss
    ┌────┴────┐
    │   DB    │  ← L3: 数据库
    └─────────┘
```

### 2. Redis 配置优化

```yaml
# Redis 配置文件优化
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  
  # 连接池配置
  pool_size: 100              # 连接池大小
  min_idle_conns: 10          # 最小空闲连接
  max_retries: 3              # 最大重试次数
  dial_timeout: "5s"          # 连接超时
  read_timeout: "3s"          # 读取超时
  write_timeout: "3s"         # 写入超时
  pool_timeout: "4s"          # 连接池获取超时
  idle_timeout: "30m"         # 空闲连接超时
  idle_check_frequency: "10m" # 空闲检查频率
```

### 3. 缓存策略实现

```go
// cache/redis_cache.go
package cache

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type Cache interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) error
    Exists(ctx context.Context, key string) (bool, error)
}

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
    return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
    data, err := c.client.Get(ctx, key).Bytes()
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.client.Set(ctx, key, data, ttl).Err()
}

// 多级缓存实现
type MultiLevelCache struct {
    local  *LocalCache      // 本地缓存
    remote *RedisCache      // Redis 缓存
    ttl    time.Duration
}

func (c *MultiLevelCache) Get(ctx context.Context, key string, dest interface{}) error {
    // 1. 尝试本地缓存
    if err := c.local.Get(key, dest); err == nil {
        return nil
    }
    
    // 2. 尝试 Redis 缓存
    if err := c.remote.Get(ctx, key, dest); err == nil {
        // 回填本地缓存
        c.local.Set(key, dest, c.ttl/10)
        return nil
    }
    
    return ErrCacheMiss
}
```

### 4. 缓存模式

```go
// 1. Cache-Aside 模式
func (s *Service) GetUser(ctx context.Context, userID int64) (*User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    // 尝试从缓存获取
    var user User
    if err := s.cache.Get(ctx, cacheKey, &user); err == nil {
        return &user, nil
    }
    
    // 从数据库获取
    user, err := s.db.GetUser(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    s.cache.Set(ctx, cacheKey, user, 30*time.Minute)
    
    return &user, nil
}

// 2. Write-Through 模式
func (s *Service) UpdateUser(ctx context.Context, user *User) error {
    // 先更新数据库
    if err := s.db.UpdateUser(ctx, user); err != nil {
        return err
    }
    
    // 再更新缓存
    cacheKey := fmt.Sprintf("user:%d", user.ID)
    s.cache.Set(ctx, cacheKey, user, 30*time.Minute)
    
    return nil
}

// 3. Write-Behind 模式（异步写入）
func (s *Service) UpdateUserAsync(ctx context.Context, user *User) error {
    // 先更新缓存
    cacheKey := fmt.Sprintf("user:%d", user.ID)
    s.cache.Set(ctx, cacheKey, user, 30*time.Minute)
    
    // 异步写入数据库
    s.asyncQueue.Publish("user.update", user)
    
    return nil
}
```

### 5. 缓存穿透、击穿、雪崩防护

```go
// 1. 缓存穿透防护 - 布隆过滤器
type BloomFilter struct {
    bitset []bool
    size   uint
    hashFuncs []func(string) uint
}

func (bf *BloomFilter) Add(item string) {
    for _, hashFunc := range bf.hashFuncs {
        index := hashFunc(item) % bf.size
        bf.bitset[index] = true
    }
}

func (bf *BloomFilter) MightContain(item string) bool {
    for _, hashFunc := range bf.hashFuncs {
        index := hashFunc(item) % bf.size
        if !bf.bitset[index] {
            return false
        }
    }
    return true
}

// 2. 缓存击穿防护 - 互斥锁
func (s *Service) GetWithMutex(ctx context.Context, key string) (*Data, error) {
    // 尝试获取缓存
    var data Data
    if err := s.cache.Get(ctx, key, &data); err == nil {
        return &data, nil
    }
    
    // 获取分布式锁
    lockKey := "lock:" + key
    acquired, err := s.redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
    if err != nil || !acquired {
        // 等待后重试
        time.Sleep(100 * time.Millisecond)
        return s.GetWithMutex(ctx, key)
    }
    
    // 双重检查
    if err := s.cache.Get(ctx, key, &data); err == nil {
        s.redisClient.Del(ctx, lockKey)
        return &data, nil
    }
    
    // 从数据库获取
    data, err = s.loadFromDB(key)
    if err != nil {
        s.redisClient.Del(ctx, lockKey)
        return nil, err
    }
    
    // 写入缓存
    s.cache.Set(ctx, key, data, 30*time.Minute)
    s.redisClient.Del(ctx, lockKey)
    
    return &data, nil
}

// 3. 缓存雪崩防护 - 随机过期时间
func (s *Service) SetWithRandomTTL(ctx context.Context, key string, value interface{}, baseTTL time.Duration) error {
    // 添加随机偏移 (0-10%)
    jitter := time.Duration(rand.Intn(10)) * baseTTL / 100
    ttl := baseTTL + jitter
    return s.cache.Set(ctx, key, value, ttl)
}
```

## 连接池配置

### 1. HTTP 连接池

```go
// 配置 HTTP 客户端连接池
func NewHTTPClient() *http.Client {
    return &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            // 连接池配置
            MaxIdleConns:        100,              // 最大空闲连接数
            MaxIdleConnsPerHost: 10,               // 每个主机的最大空闲连接
            MaxConnsPerHost:     100,              // 每个主机的最大连接数
            IdleConnTimeout:     90 * time.Second, // 空闲连接超时
            
            // TLS 配置
            TLSHandshakeTimeout:   10 * time.Second,
            ExpectContinueTimeout: 1 * time.Second,
            
            // 连接复用
            DisableKeepAlives: false,
            DisableCompression: false,
            
            // 连接建立
            DialContext: (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext,
        },
    }
}
```

### 2. gRPC 连接池

```go
// gRPC 连接池
func NewGRPCConnPool(address string, poolSize int) (*GRPCPool, error) {
    pool := &GRPCPool{
        conns: make(chan *grpc.ClientConn, poolSize),
    }
    
    for i := 0; i < poolSize; i++ {
        conn, err := grpc.Dial(address,
            grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
            grpc.WithKeepaliveParams(keepalive.ClientParameters{
                Time:                10 * time.Second,
                Timeout:             3 * time.Second,
                PermitWithoutStream: true,
            }),
        )
        if err != nil {
            return nil, err
        }
        pool.conns <- conn
    }
    
    return pool, nil
}
```

### 3. 数据库连接池监控

```go
// 监控连接池状态
func MonitorDBPool(db *sql.DB) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := db.Stats()
        
        metrics.DBPoolStats.WithLabelValues("open").Set(float64(stats.OpenConnections))
        metrics.DBPoolStats.WithLabelValues("in_use").Set(float64(stats.InUse))
        metrics.DBPoolStats.WithLabelValues("idle").Set(float64(stats.Idle))
        metrics.DBPoolStats.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
        metrics.DBPoolStats.WithLabelValues("wait_duration").Set(stats.WaitDuration.Seconds())
        
        // 告警：连接池使用率过高
        if stats.OpenConnections > int(float64(stats.MaxOpenConnections)*0.8) {
            alerts.Send("DB connection pool usage > 80%")
        }
    }
}
```

## 负载均衡

### 1. 服务端负载均衡

```yaml
# nginx.conf
upstream backend {
    least_conn;                    # 最少连接算法
    server backend-1:8080 weight=5;
    server backend-2:8080 weight=5;
    server backend-3:8080 weight=5 backup;
    
    keepalive 100;                 # 长连接数
    keepalive_timeout 60s;
    keepalive_requests 1000;
}

server {
    listen 80;
    
    location / {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        
        # 超时配置
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # 缓冲区
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        
        # 健康检查
        health_check interval=5s fails=3 passes=2;
    }
}
```

### 2. 客户端负载均衡

```go
// 客户端负载均衡器
type LoadBalancer struct {
    backends []*Backend
    strategy Strategy
    mu       sync.RWMutex
}

type Backend struct {
    Address string
    Weight  int
    Healthy bool
    Conns   int32
}

// 轮询策略
func (lb *LoadBalancer) RoundRobin() *Backend {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    var selected *Backend
    minConns := int32(^uint32(0) >> 1)
    
    for _, b := range lb.backends {
        if !b.Healthy {
            continue
        }
        if atomic.LoadInt32(&b.Conns) < minConns {
            minConns = atomic.LoadInt32(&b.Conns)
            selected = b
        }
    }
    
    if selected != nil {
        atomic.AddInt32(&selected.Conns, 1)
    }
    
    return selected
}

// 加权轮询
func (lb *LoadBalancer) WeightedRoundRobin() *Backend {
    // 实现平滑加权轮询算法
    // ...
}
```

### 3. 健康检查

```go
// 健康检查器
type HealthChecker struct {
    backends []*Backend
    interval time.Duration
    timeout  time.Duration
}

func (hc *HealthChecker) Start() {
    ticker := time.NewTicker(hc.interval)
    defer ticker.Stop()
    
    for range ticker.C {
        for _, backend := range hc.backends {
            go hc.check(backend)
        }
    }
}

func (hc *HealthChecker) check(backend *Backend) {
    ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", 
        backend.Address+"/health", nil)
    
    resp, err := http.DefaultClient.Do(req)
    healthy := err == nil && resp.StatusCode == 200
    
    backend.mu.Lock()
    backend.Healthy = healthy
    backend.mu.Unlock()
}
```

## 监控指标

### 1. 应用指标

```go
// metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP 请求指标
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    
    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
    
    // 业务指标
    RouterRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "router_requests_total",
            Help: "Total router requests by provider",
        },
        []string{"provider", "model"},
    )
    
    RouterLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "router_latency_seconds",
            Help:    "Router latency",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"provider"},
    )
    
    // 数据库指标
    DBQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )
    
    DBPoolStats = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "db_pool_stats",
            Help: "Database pool statistics",
        },
        []string{"type"},
    )
    
    // 缓存指标
    CacheHits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Total cache hits",
        },
        []string{"cache_type"},
    )
    
    CacheMisses = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_misses_total",
            Help: "Total cache misses",
        },
        []string{"cache_type"},
    )
)
```

### 2. 中间件实现

```go
// middleware/metrics.go
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        path := c.FullPath()
        method := c.Request.Method
        
        metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
        metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
    }
}
```

### 3. Grafana Dashboard

```json
{
  "dashboard": {
    "title": "MaaS Router Performance",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [{
          "expr": "rate(http_requests_total[5m])",
          "legendFormat": "{{method}} {{status}}"
        }]
      },
      {
        "title": "Latency (P50/P95/P99)",
        "targets": [
          {"expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))", "legendFormat": "P50"},
          {"expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))", "legendFormat": "P95"},
          {"expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))", "legendFormat": "P99"}
        ]
      },
      {
        "title": "Cache Hit Rate",
        "targets": [{
          "expr": "rate(cache_hits_total[5m]) / (rate(cache_hits_total[5m]) + rate(cache_misses_total[5m]))",
          "legendFormat": "{{cache_type}}"
        }]
      },
      {
        "title": "DB Connection Pool",
        "targets": [
          {"expr": "db_pool_stats{type=\"open\"}", "legendFormat": "Open"},
          {"expr": "db_pool_stats{type=\"in_use\"}", "legendFormat": "In Use"},
          {"expr": "db_pool_stats{type=\"idle\"}", "legendFormat": "Idle"}
        ]
      }
    ]
  }
}
```

## 性能测试

### 1. 压力测试

```bash
# 使用 vegeta 进行压力测试
echo "GET http://localhost:8080/health" | vegeta attack -rate=1000 -duration=60s | vegeta report

# 使用 wrk 进行压力测试
wrk -t12 -c400 -d60s http://localhost:8080/health

# 使用 k6 进行脚本化测试
k6 run --vus 100 --duration 60s load-test.js
```

### 2. 性能基准

| 指标 | 目标值 | 说明 |
|------|--------|------|
| P50 延迟 | < 50ms | 50% 请求响应时间 |
| P95 延迟 | < 200ms | 95% 请求响应时间 |
| P99 延迟 | < 500ms | 99% 请求响应时间 |
| 吞吐量 | > 10,000 RPS | 每秒请求数 |
| 错误率 | < 0.1% | HTTP 5xx 错误比例 |
| 缓存命中率 | > 90% | Redis 缓存命中率 |
| DB 连接使用率 | < 80% | 数据库连接池使用率 |
