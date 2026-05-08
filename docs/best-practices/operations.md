# MaaS-Router 运维最佳实践

本文档介绍 MaaS-Router 项目的运维最佳实践，包括备份策略、升级流程、故障排查、日志分析和容量规划。

## 目录

- [备份策略](#备份策略)
- [升级流程](#升级流程)
- [故障排查](#故障排查)
- [日志分析](#日志分析)
- [容量规划](#容量规划)

## 备份策略

### 1. 数据库备份

#### PostgreSQL 备份方案

```bash
#!/bin/bash
# backup/postgres-backup.sh

set -e

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-maas_user}
DB_NAME=${DB_NAME:-maas_router}
BACKUP_DIR=${BACKUP_DIR:-/backup/postgres}
RETENTION_DAYS=${RETENTION_DAYS:-30}

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/maas_router_${DATE}.sql"

# 创建备份目录
mkdir -p ${BACKUP_DIR}

# 执行备份
echo "Starting PostgreSQL backup..."
pg_dump -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} \
    --format=custom \
    --compress=9 \
    --verbose \
    --file=${BACKUP_FILE}.dump

# 同时生成 SQL 格式备份（用于部分恢复）
pg_dump -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} \
    --format=plain \
    --verbose \
    --file=${BACKUP_FILE}.sql

# 压缩 SQL 备份
gzip ${BACKUP_FILE}.sql

# 计算校验和
cd ${BACKUP_DIR}
sha256sum maas_router_${DATE}.sql.gz > maas_router_${DATE}.sha256
sha256sum maas_router_${DATE}.dump >> maas_router_${DATE}.sha256

# 上传到对象存储（AWS S3）
aws s3 cp ${BACKUP_FILE}.dump s3://maas-router-backups/postgres/
aws s3 cp ${BACKUP_FILE}.sql.gz s3://maas-router-backups/postgres/
aws s3 cp ${BACKUP_DIR}/maas_router_${DATE}.sha256 s3://maas-router-backups/postgres/

# 清理旧备份
echo "Cleaning up old backups..."
find ${BACKUP_DIR} -name "maas_router_*.dump" -mtime +${RETENTION_DAYS} -delete
find ${BACKUP_DIR} -name "maas_router_*.sql.gz" -mtime +${RETENTION_DAYS} -delete
find ${BACKUP_DIR} -name "maas_router_*.sha256" -mtime +${RETENTION_DAYS} -delete

# 删除 S3 上的旧备份
aws s3 ls s3://maas-router-backups/postgres/ | awk '{print $4}' | while read file; do
    file_date=$(echo $file | grep -oP '\d{8}_\d{6}')
    if [ ! -z "$file_date" ]; then
        file_timestamp=$(date -d "${file_date:0:8} ${file_date:9:2}:${file_date:11:2}:${file_date:13:2}" +%s)
        current_timestamp=$(date +%s)
        age_days=$(( (current_timestamp - file_timestamp) / 86400 ))
        if [ $age_days -gt ${RETENTION_DAYS} ]; then
            aws s3 rm s3://maas-router-backups/postgres/$file
        fi
    fi
done

echo "Backup completed: ${BACKUP_FILE}"
```

#### 定时备份任务

```yaml
# kubernetes CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: maas-router
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16-alpine
            command:
            - /bin/sh
            - -c
            - |
              export PGPASSWORD=$(cat /secrets/db-password)
              pg_dump -h postgres -U maas_user -d maas_router \
                --format=custom --file=/backup/maas_router_$(date +%Y%m%d).dump
              
              # 上传到 S3
              aws s3 cp /backup/maas_router_$(date +%Y%m%d).dump \
                s3://maas-router-backups/postgres/
            env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: access-key-id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: aws-credentials
                  key: secret-access-key
            volumeMounts:
            - name: backup
              mountPath: /backup
            - name: secrets
              mountPath: /secrets
          volumes:
          - name: backup
            emptyDir: {}
          - name: secrets
            secret:
              secretName: postgres-backup-secrets
          restartPolicy: OnFailure
```

### 2. Redis 备份

```bash
#!/bin/bash
# backup/redis-backup.sh

REDIS_HOST=${REDIS_HOST:-localhost}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_PASSWORD=${REDIS_PASSWORD:-}
BACKUP_DIR=${BACKUP_DIR:-/backup/redis}
RETENTION_DAYS=${RETENTION_DAYS:-30}

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/redis_${DATE}.rdb"

mkdir -p ${BACKUP_DIR}

# 触发 BGSAVE
if [ -z "$REDIS_PASSWORD" ]; then
    redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT} BGSAVE
else
    redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT} -a ${REDIS_PASSWORD} BGSAVE
fi

# 等待保存完成
sleep 5

# 复制 RDB 文件
kubectl cp maas-router/redis-0:/data/dump.rdb ${BACKUP_FILE}

# 压缩
gzip ${BACKUP_FILE}

# 上传到 S3
aws s3 cp ${BACKUP_FILE}.gz s3://maas-router-backups/redis/

# 清理旧备份
find ${BACKUP_DIR} -name "redis_*.rdb.gz" -mtime +${RETENTION_DAYS} -delete
```

### 3. 配置文件备份

```bash
#!/bin/bash
# backup/config-backup.sh

BACKUP_DIR=${BACKUP_DIR:-/backup/config}
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p ${BACKUP_DIR}

# 备份 Kubernetes 配置
kubectl get configmap -n maas-router -o yaml > ${BACKUP_DIR}/configmaps_${DATE}.yaml
kubectl get secret -n maas-router -o yaml > ${BACKUP_DIR}/secrets_${DATE}.yaml
kubectl get ingress -n maas-router -o yaml > ${BACKUP_DIR}/ingress_${DATE}.yaml

# 加密敏感数据
gpg --encrypt --recipient ops@maas-router.com \
    ${BACKUP_DIR}/secrets_${DATE}.yaml
rm ${BACKUP_DIR}/secrets_${DATE}.yaml

# 压缩
tar -czf ${BACKUP_DIR}/k8s_config_${DATE}.tar.gz \
    ${BACKUP_DIR}/configmaps_${DATE}.yaml \
    ${BACKUP_DIR}/secrets_${DATE}.yaml.gpg \
    ${BACKUP_DIR}/ingress_${DATE}.yaml

# 清理临时文件
rm ${BACKUP_DIR}/configmaps_${DATE}.yaml
rm ${BACKUP_DIR}/secrets_${DATE}.yaml.gpg
rm ${BACKUP_DIR}/ingress_${DATE}.yaml

# 上传到 S3
aws s3 cp ${BACKUP_DIR}/k8s_config_${DATE}.tar.gz s3://maas-router-backups/config/
```

### 4. 恢复流程

```bash
#!/bin/bash
# restore/postgres-restore.sh

set -e

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-maas_user}
DB_NAME=${DB_NAME:-maas_router}
BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# 下载备份（如果在 S3）
if [[ $BACKUP_FILE == s3://* ]]; then
    aws s3 cp ${BACKUP_FILE} /tmp/restore.dump
    BACKUP_FILE=/tmp/restore.dump
fi

# 验证备份文件
if [ ! -f "$BACKUP_FILE" ]; then
    echo "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# 创建新数据库（用于恢复测试）
RESTORE_DB="${DB_NAME}_restore_$(date +%s)"
echo "Creating restore database: ${RESTORE_DB}"
psql -h ${DB_HOST} -p ${DB_PORT} -U postgres -c "CREATE DATABASE ${RESTORE_DB};"

# 恢复数据
echo "Restoring data..."
pg_restore -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${RESTORE_DB} \
    --verbose --no-owner --no-privileges ${BACKUP_FILE}

# 验证恢复
echo "Verifying restore..."
TABLE_COUNT=$(psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${RESTORE_DB} \
    -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")
echo "Restored tables: ${TABLE_COUNT}"

# 切换数据库（生产环境谨慎操作）
read -p "Switch to restored database? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    psql -h ${DB_HOST} -p ${DB_PORT} -U postgres <<EOF
    SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${DB_NAME}';
    DROP DATABASE ${DB_NAME};
    ALTER DATABASE ${RESTORE_DB} RENAME TO ${DB_NAME};
EOF
    echo "Database switched successfully"
fi
```

## 升级流程

### 1. 升级前检查清单

```bash
#!/bin/bash
# upgrade/pre-upgrade-check.sh

echo "=== Pre-Upgrade Checklist ==="

# 1. 检查当前版本
echo "Current version:"
kubectl get deployment api-gateway -n maas-router -o jsonpath='{.spec.template.spec.containers[0].image}'
echo

# 2. 检查系统健康
echo "Checking system health..."
kubectl get pods -n maas-router

# 3. 检查资源使用
echo "Resource usage:"
kubectl top pods -n maas-router

# 4. 检查数据库连接
echo "Database connections:"
kubectl exec -it postgres-0 -n maas-router -- psql -U maas_user -c "SELECT count(*) FROM pg_stat_activity;"

# 5. 备份当前配置
echo "Backing up current configuration..."
kubectl get deployment api-gateway -n maas-router -o yaml > /backup/pre-upgrade-deployment.yaml

# 6. 检查新版本兼容性
echo "Checking migration scripts..."
ls -la migrations/

# 7. 验证镜像
echo "Verifying new image..."
docker pull maas-router/backend:v1.1.0

echo "Pre-upgrade check completed!"
```

### 2. 蓝绿部署升级

```yaml
# upgrade/blue-green-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway-green
  namespace: maas-router
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
      version: green
  template:
    metadata:
      labels:
        app: api-gateway
        version: green
    spec:
      containers:
      - name: api-gateway
        image: maas-router/backend:v1.1.0  # 新版本
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: maas-secrets
              key: database-url
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: api-gateway-green
  namespace: maas-router
spec:
  selector:
    app: api-gateway
    version: green
  ports:
  - port: 80
    targetPort: 8080
```

```bash
#!/bin/bash
# upgrade/blue-green-switch.sh

# 1. 部署绿色环境
echo "Deploying green environment..."
kubectl apply -f upgrade/blue-green-deployment.yaml

# 2. 等待绿色环境就绪
echo "Waiting for green environment to be ready..."
kubectl rollout status deployment/api-gateway-green -n maas-router --timeout=300s

# 3. 验证绿色环境
echo "Verifying green environment..."
GREEN_POD=$(kubectl get pod -l app=api-gateway,version=green -n maas-router -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it ${GREEN_POD} -n maas-router -- wget -qO- http://localhost:8080/health

# 4. 切换流量
echo "Switching traffic to green..."
kubectl patch service api-gateway -n maas-router -p '{"spec":{"selector":{"version":"green"}}}'

# 5. 监控一段时间
echo "Monitoring for 5 minutes..."
sleep 300

# 6. 如果一切正常，删除蓝色环境
read -p "Remove blue environment? (yes/no): " CONFIRM
if [ "$CONFIRM" = "yes" ]; then
    kubectl delete deployment api-gateway -n maas-router
    echo "Blue environment removed"
fi
```

### 3. 数据库迁移

```go
// migrations/migration.go
package migrations

import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) error {
    m, err := migrate.New(
        "file://migrations",
        databaseURL,
    )
    if err != nil {
        return err
    }
    
    // 执行迁移
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    
    version, dirty, err := m.Version()
    if err != nil {
        return err
    }
    
    log.Printf("Migration completed: version=%d, dirty=%v", version, dirty)
    return nil
}

func RollbackMigration(databaseURL string) error {
    m, err := migrate.New(
        "file://migrations",
        databaseURL,
    )
    if err != nil {
        return err
    }
    
    // 回滚一步
    if err := m.Steps(-1); err != nil {
        return err
    }
    
    return nil
}
```

## 故障排查

### 1. 常见故障诊断流程

```bash
#!/bin/bash
# troubleshooting/diagnose.sh

echo "=== MaaS-Router 故障诊断 ==="

# 1. 检查 Pod 状态
echo "1. Pod Status:"
kubectl get pods -n maas-router

# 2. 检查事件
echo -e "\n2. Recent Events:"
kubectl get events -n maas-router --sort-by='.lastTimestamp' | tail -20

# 3. 检查资源使用
echo -e "\n3. Resource Usage:"
kubectl top pods -n maas-router

# 4. 检查服务状态
echo -e "\n4. Service Status:"
kubectl get svc -n maas-router
kubectl get endpoints -n maas-router

# 5. 检查 Ingress
echo -e "\n5. Ingress Status:"
kubectl get ingress -n maas-router

# 6. 检查日志
echo -e "\n6. Recent Logs:"
BACKEND_POD=$(kubectl get pod -l app=api-gateway -n maas-router -o jsonpath='{.items[0].metadata.name}')
kubectl logs ${BACKEND_POD} -n maas-router --tail=50

# 7. 数据库连接检查
echo -e "\n7. Database Connection:"
kubectl exec -it postgres-0 -n maas-router -- pg_isready -U maas_user

# 8. Redis 连接检查
echo -e "\n8. Redis Connection:"
kubectl exec -it redis-0 -n maas-router -- redis-cli ping
```

### 2. 性能问题排查

```bash
#!/bin/bash
# troubleshooting/performance-check.sh

echo "=== Performance Diagnostics ==="

# 1. 慢查询检查
echo "1. Slow Queries:"
kubectl exec -it postgres-0 -n maas-router -- psql -U maas_user -c "
SELECT query, calls, mean_exec_time, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
"

# 2. 数据库连接数
echo -e "\n2. Database Connections:"
kubectl exec -it postgres-0 -n maas-router -- psql -U maas_user -c "
SELECT state, count(*)
FROM pg_stat_activity
GROUP BY state;
"

# 3. 锁等待检查
echo -e "\n3. Lock Waits:"
kubectl exec -it postgres-0 -n maas-router -- psql -U maas_user -c "
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
"

# 4. Redis 内存使用
echo -e "\n4. Redis Memory:"
kubectl exec -it redis-0 -n maas-router -- redis-cli info memory | grep used_memory

# 5. 应用指标
echo -e "\n5. Application Metrics:"
curl -s http://localhost:8080/metrics | grep -E "http_request_duration|goroutines"
```

### 3. 网络问题排查

```bash
#!/bin/bash
# troubleshooting/network-check.sh

echo "=== Network Diagnostics ==="

# 1. DNS 解析检查
echo "1. DNS Resolution:"
kubectl run -it --rm debug --image=busybox:1.36 --restart=Never -- nslookup postgres.maas-router.svc.cluster.local

# 2. 服务连通性
echo -e "\n2. Service Connectivity:"
BACKEND_POD=$(kubectl get pod -l app=api-gateway -n maas-router -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it ${BACKEND_POD} -n maas-router -- wget -qO- http://postgres:5432 || echo "PostgreSQL connection check"

# 3. Ingress 检查
echo -e "\n3. Ingress Controller:"
kubectl get pods -n ingress-nginx
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx --tail=20

# 4. 网络策略检查
echo -e "\n4. Network Policies:"
kubectl get networkpolicies -n maas-router
```

## 日志分析

### 1. 日志收集架构

```yaml
# logging/fluentd-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
  namespace: logging
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/*.log
      pos_file /var/log/fluentd-containers.log.pos
      tag kubernetes.*
      <parse>
        @type json
        time_key time
        time_format %Y-%m-%dT%H:%M:%S.%NZ
      </parse>
    </source>
    
    <filter kubernetes.**>
      @type kubernetes_metadata
    </filter>
    
    <filter kubernetes.**>
      @type grep
      <regexp>
        key $.kubernetes.labels.app
        pattern ^api-gateway|judge-agent|postgres|redis$
      </regexp>
    </filter>
    
    <match kubernetes.**>
      @type elasticsearch
      host elasticsearch
      port 9200
      logstash_format true
      logstash_prefix maas-router
      <buffer>
        flush_interval 10s
      </buffer>
    </match>
```

### 2. 日志查询示例

```bash
# 查询特定用户的请求
kubectl logs -l app=api-gateway -n maas-router --tail=1000 | jq 'select(.user_id == 123)'

# 查询错误日志
kubectl logs -l app=api-gateway -n maas-router | jq 'select(.level == "error")'

# 查询慢请求
kubectl logs -l app=api-gateway -n maas-router | jq 'select(.latency_ms > 1000)'

# 使用 Loki 查询
logcli query '{namespace="maas-router", app="api-gateway"} |= "error"'
```

### 3. 日志告警规则

```yaml
# logging/log-alerts.yaml
groups:
  - name: log-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate({namespace="maas-router"} |= "error" [5m])) 
          / sum(rate({namespace="maas-router"} [5m])) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in logs"
          
      - alert: DatabaseConnectionError
        expr: |
          sum(rate({namespace="maas-router"} |= "connection refused" [5m])) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database connection errors detected"
```

## 容量规划

### 1. 资源使用监控

```yaml
# monitoring/resource-dashboard.json
{
  "dashboard": {
    "title": "Capacity Planning",
    "panels": [
      {
        "title": "CPU Usage Trend",
        "targets": [{
          "expr": "rate(container_cpu_usage_seconds_total{namespace=\"maas-router\"}[5m])",
          "legendFormat": "{{pod}}"
        }]
      },
      {
        "title": "Memory Usage Trend",
        "targets": [{
          "expr": "container_memory_usage_bytes{namespace=\"maas-router\"}",
          "legendFormat": "{{pod}}"
        }]
      },
      {
        "title": "Request Rate",
        "targets": [{
          "expr": "rate(http_requests_total{namespace=\"maas-router\"}[5m])",
          "legendFormat": "{{pod}}"
        }]
      },
      {
        "title": "Database Connections",
        "targets": [{
          "expr": "pg_stat_activity_count{namespace=\"maas-router\"}",
          "legendFormat": "Connections"
        }]
      }
    ]
  }
}
```

### 2. 容量规划公式

```python
# capacity/planning.py

class CapacityPlanner:
    def __init__(self):
        self.current_rps = 1000  # 当前每秒请求数
        self.current_pods = 3
        self.current_cpu_per_pod = 0.5  # 每个 Pod CPU 核心数
        self.current_memory_per_pod = 512  # MB
        
    def calculate_required_resources(self, target_rps, growth_factor=1.5):
        """
        计算所需资源
        
        target_rps: 目标每秒请求数
        growth_factor: 增长系数（预留空间）
        """
        # 计算需要的 Pod 数量
        rps_per_pod = self.current_rps / self.current_pods
        required_pods = (target_rps / rps_per_pod) * growth_factor
        
        # 计算总资源需求
        total_cpu = required_pods * self.current_cpu_per_pod
        total_memory = required_pods * self.current_memory_per_pod
        
        return {
            'pods': int(required_pods),
            'cpu_cores': total_cpu,
            'memory_mb': total_memory,
            'memory_gb': total_memory / 1024
        }
    
    def estimate_database_capacity(self, daily_active_users, avg_requests_per_user):
        """
        估算数据库容量需求
        """
        # 日均请求数
        daily_requests = daily_active_users * avg_requests_per_user
        
        # 存储估算（每条请求日志约 2KB）
        daily_storage_gb = (daily_requests * 2 * 1024) / (1024**3)
        
        # 考虑保留期（1年）
        yearly_storage_gb = daily_storage_gb * 365
        
        # 考虑索引和开销（2倍）
        total_storage_gb = yearly_storage_gb * 2
        
        # 连接数估算
        # 假设每个活跃用户每10秒产生一个请求，连接池复用
        concurrent_users = daily_active_users * 0.1  # 10% 同时在线
        required_connections = min(concurrent_users, 500)  # 最大500连接
        
        return {
            'daily_requests': daily_requests,
            'daily_storage_gb': daily_storage_gb,
            'yearly_storage_gb': yearly_storage_gb,
            'total_storage_gb': total_storage_gb,
            'required_connections': int(required_connections)
        }
    
    def estimate_redis_capacity(self, cache_hit_rate=0.9, avg_object_size_kb=10):
        """
        估算 Redis 容量需求
        """
        # 活跃缓存对象数
        active_objects = self.current_rps * 60 * 10  # 10分钟窗口
        
        # 内存需求
        memory_mb = (active_objects * avg_object_size_kb) / 1024
        
        # 考虑缓存命中率
        effective_memory = memory_mb / cache_hit_rate
        
        return {
            'active_objects': active_objects,
            'memory_mb': effective_memory,
            'memory_gb': effective_memory / 1024,
            'recommended_cluster_size': max(1, int(effective_memory / 2048))  # 每2GB一个分片
        }

# 使用示例
planner = CapacityPlanner()

# 规划支持 10000 RPS 的资源
resources = planner.calculate_required_resources(10000)
print(f"Required resources for 10000 RPS:")
print(f"  Pods: {resources['pods']}")
print(f"  CPU: {resources['cpu_cores']:.1f} cores")
print(f"  Memory: {resources['memory_gb']:.1f} GB")

# 数据库容量规划
db_capacity = planner.estimate_database_capacity(
    daily_active_users=100000,
    avg_requests_per_user=100
)
print(f"\nDatabase capacity for 100K DAU:")
print(f"  Daily requests: {db_capacity['daily_requests']:,}")
print(f"  Yearly storage: {db_capacity['yearly_storage_gb']:.1f} GB")
print(f"  Total storage: {db_capacity['total_storage_gb']:.1f} GB")
print(f"  Required connections: {db_capacity['required_connections']}")
```

### 3. 自动扩缩容策略

```yaml
# capacity/hpa-config.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
  namespace: maas-router
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 50
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "1000"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 100
          periodSeconds: 60
        - type: Pods
          value: 4
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
```

### 4. 容量规划检查清单

- [ ] 当前资源使用率监控（CPU、内存、磁盘、网络）
- [ ] 历史增长趋势分析
- [ ] 峰值流量预测
- [ ] 数据库容量增长预测
- [ ] 存储容量规划（日志、备份）
- [ ] 网络带宽需求评估
- [ ] 自动扩缩容策略配置
- [ ] 成本估算和预算
- [ ] 灾难恢复容量预留
- [ ] 定期容量审查（每月/每季度）

## 运维自动化脚本

### 1. 健康检查脚本

```bash
#!/bin/bash
# ops/health-check.sh

HEALTH_STATUS=0

check_service() {
    local name=$1
    local url=$2
    
    if curl -sf ${url} > /dev/null 2>&1; then
        echo "[OK] ${name}"
    else
        echo "[FAIL] ${name}"
        HEALTH_STATUS=1
    fi
}

echo "=== Health Check ==="
check_service "API Gateway" "http://api.maas-router.com/health"
check_service "Database" "http://api.maas-router.com/ready"
check_service "Prometheus" "http://prometheus.maas-router.com/-/healthy"
check_service "Grafana" "http://grafana.maas-router.com/api/health"

exit ${HEALTH_STATUS}
```

### 2. 日常巡检脚本

```bash
#!/bin/bash
# ops/daily-check.sh

REPORT_FILE="/var/log/maas-router/daily-check-$(date +%Y%m%d).log"

exec > >(tee -a ${REPORT_FILE})
exec 2>&1

echo "=== Daily Check Report - $(date) ==="

# 系统资源
echo -e "\n## System Resources"
df -h | grep -E "Filesystem|/data|/var"
free -h

# Kubernetes 状态
echo -e "\n## Kubernetes Status"
kubectl get nodes
kubectl get pods -n maas-router

# 备份状态
echo -e "\n## Backup Status"
ls -lh /backup/postgres/ | tail -5

# 证书过期检查
echo -e "\n## Certificate Expiry"
kubectl get certificates -n maas-router -o json | jq -r '.items[] | "\(.metadata.name): \(.status.notAfter)"'

# 发送报告
mail -s "MaaS-Router Daily Check - $(date +%Y-%m-%d)" ops@maas-router.com < ${REPORT_FILE}
```

以上所有文档已创建完成。让我更新任务状态并总结。