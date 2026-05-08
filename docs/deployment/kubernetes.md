# MaaS-Router Kubernetes 部署指南

本文档介绍如何使用 Kubernetes 部署 MaaS-Router 项目，包括 Helm Chart 安装、手动部署、自动扩缩容和监控告警配置。

## 目录

- [环境要求](#环境要求)
- [Helm Chart 安装](#helm-chart-安装)
- [手动部署步骤](#手动部署步骤)
- [配置 ConfigMap 和 Secret](#配置-configmap-和-secret)
- [HPA 自动扩缩容](#hpa-自动扩缩容)
- [监控和告警](#监控和告警)
- [故障排查](#故障排查)

## 环境要求

- Kubernetes 1.25+
- kubectl 1.25+
- Helm 3.0+（可选）
- 至少 4 个节点，每个节点 4GB+ 内存
- 存储类（StorageClass）已配置

## Helm Chart 安装

### 1. 添加 Helm 仓库

```bash
# 添加 MaaS-Router Helm 仓库
helm repo add maas-router https://charts.maas-router.io
helm repo update
```

### 2. 安装 Chart

```bash
# 创建命名空间
kubectl create namespace maas-router

# 安装基础版本
helm install maas-router maas-router/maas-router \
  --namespace maas-router \
  --set global.domain=api.maas-router.com

# 安装生产版本（自定义配置）
helm install maas-router maas-router/maas-router \
  --namespace maas-router \
  -f values-production.yaml
```

### 3. 自定义 values.yaml

```yaml
# values-production.yaml
global:
  domain: api.maas-router.com
  environment: production

apiGateway:
  replicas: 3
  resources:
    requests:
      cpu: 1000m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 2Gi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80

postgres:
  enabled: true
  persistence:
    size: 100Gi
    storageClass: fast-ssd
  resources:
    requests:
      cpu: 1000m
      memory: 2Gi
    limits:
      cpu: 2000m
      memory: 4Gi

redis:
  enabled: true
  cluster:
    enabled: true
    slaveCount: 2
  persistence:
    size: 20Gi

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
  hosts:
    - host: api.maas-router.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: api-gateway-tls
      hosts:
        - api.maas-router.com

monitoring:
  enabled: true
  prometheus:
    retention: 30d
  grafana:
    enabled: true
    adminUser: admin
    adminPassword: admin123
```

### 4. 升级和卸载

```bash
# 升级
helm upgrade maas-router maas-router/maas-router \
  --namespace maas-router \
  -f values-production.yaml

# 卸载
helm uninstall maas-router --namespace maas-router
```

## 手动部署步骤

### 1. 创建命名空间

```bash
kubectl apply -f infra/k8s/namespace.yaml
```

### 2. 配置 Secrets

```bash
# 创建 Secret（生产环境请使用加密工具）
kubectl create secret generic maas-secrets \
  --namespace maas-router \
  --from-literal=database-url="postgres://maas_user:secure_password@postgres:5432/maas_router?sslmode=require" \
  --from-literal=postgres-user="maas_user" \
  --from-literal=postgres-password="secure_password" \
  --from-literal=redis-url="redis://redis-cluster:6379" \
  --from-literal=jwt-secret="your-super-secret-jwt-key-change-this-in-production" \
  --from-literal=deepseek-api-key="sk-xxxxxxxxxxxxxxxx" \
  --from-literal=grafana-admin-user="admin" \
  --from-literal=grafana-admin-password="admin123"
```

或使用 YAML 文件：

```bash
# 编辑 secrets.yaml 后应用
kubectl apply -f infra/k8s/secrets.yaml
```

### 3. 部署数据库

```bash
# 部署 PostgreSQL
kubectl apply -f infra/k8s/postgres.yaml

# 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod -l app=postgres -n maas-router --timeout=300s

# 运行数据库迁移
kubectl apply -f infra/k8s/postgres-migrations.yaml
```

### 4. 部署 Redis

```bash
# 部署 Redis
kubectl apply -f infra/k8s/redis.yaml

# 等待 Redis 就绪
kubectl wait --for=condition=ready pod -l app=redis -n maas-router --timeout=300s
```

### 5. 部署 API Gateway

```bash
# 部署 API Gateway
kubectl apply -f infra/k8s/api-gateway.yaml

# 等待就绪
kubectl wait --for=condition=ready pod -l app=api-gateway -n maas-router --timeout=300s
```

### 6. 部署 Judge Agent

```bash
# 部署 Judge Agent
kubectl apply -f infra/k8s/judge-agent.yaml
```

### 7. 验证部署

```bash
# 查看所有 Pod
kubectl get pods -n maas-router

# 查看服务
kubectl get svc -n maas-router

# 查看 Ingress
kubectl get ingress -n maas-router
```

## 配置 ConfigMap 和 Secret

### ConfigMap 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: maas-config
  namespace: maas-router
data:
  # 服务器配置
  server.http.addr: "0.0.0.0:8080"
  server.http.timeout: "30s"
  server.grpc.addr: "0.0.0.0:9090"
  server.grpc.timeout: "30s"
  
  # 数据库配置
  database.postgres.max-conns: "100"
  database.postgres.min-conns: "10"
  database.redis.pool-size: "100"
  
  # Kafka 配置
  kafka.brokers: "kafka-0:9092,kafka-1:9092,kafka-2:9092"
  
  # 路由配置
  router.judge.endpoint: "http://judge-agent:8000"
  router.judge.timeout: "100ms"
  router.judge.fallback-to-rules: "true"
  
  # 计费配置
  billing.cred.chain-id: "137"
  billing.cred.contract-address: "0x..."
  billing.cred.settlement-hour: "0"
  
  # 日志配置
  logging.level: "info"
  logging.format: "json"
```

### Secret 配置（加密）

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: maas-secrets
  namespace: maas-router
type: Opaque
stringData:
  # 数据库
  database-url: "postgres://maas_user:secure_password@postgres:5432/maas_router?sslmode=require"
  postgres-user: "maas_user"
  postgres-password: "secure_password"
  
  # Redis
  redis-url: "redis://redis-cluster:6379"
  redis-password: ""
  
  # JWT
  jwt-secret: "your-super-secret-jwt-key-change-this-in-production"
  
  # API Keys（加密存储）
  deepseek-api-key: "sk-xxxxxxxxxxxxxxxx"
  azure-openai-api-key: "xxxxxxxxxxxxxxxx"
  anthropic-api-key: "sk-ant-xxxxxxxxxxxxxxxx"
  
  # 区块链
  private-key: "0x0000000000000000000000000000000000000000000000000000000000000000"
  
  # Kafka
  kafka-sasl-username: "maas"
  kafka-sasl-password: "kafka_password"
```

### 使用 Sealed Secrets（推荐）

```bash
# 安装 kubeseal
brew install kubeseal

# 加密 Secret
kubeseal --format=yaml < secret.yaml > sealed-secret.yaml

# 部署加密后的 Secret
kubectl apply -f sealed-secret.yaml
```

## HPA 自动扩缩容

### 1. 启用 Metrics Server

```bash
# 安装 Metrics Server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 验证
kubectl top nodes
kubectl top pods -n maas-router
```

### 2. 配置 HPA

```yaml
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
  maxReplicas: 20
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
      selectPolicy: Max
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
        - type: Pods
          value: 2
          periodSeconds: 60
      selectPolicy: Min
```

### 3. 自定义指标 HPA（使用 Prometheus Adapter）

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-custom-hpa
  namespace: maas-router
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 50
  metrics:
    - type: External
      external:
        metric:
          name: http_requests_per_second
          selector:
            matchLabels:
              service: api-gateway
        target:
          type: AverageValue
          averageValue: "1000"
    - type: External
      external:
        metric:
          name: http_request_duration_seconds
          selector:
            matchLabels:
              quantile: "0.99"
        target:
          type: Value
          value: "2"
```

### 4. 验证 HPA

```bash
# 查看 HPA 状态
kubectl get hpa -n maas-router

# 查看 HPA 详情
kubectl describe hpa api-gateway-hpa -n maas-router

# 压力测试
echo "GET http://api.maas-router.com/health" | vegeta attack -rate=1000 -duration=60s | vegeta report
```

## 监控和告警

### 1. 部署 Prometheus

```bash
# 部署 Prometheus
kubectl apply -f infra/monitoring/prometheus.yaml

# 验证
kubectl get pods -n maas-router -l app=prometheus
```

### 2. 部署 Grafana

```bash
# 部署 Grafana
kubectl apply -f infra/monitoring/grafana.yaml

# 获取 Grafana 密码
kubectl get secret maas-secrets -n maas-router -o jsonpath='{.data.grafana-admin-password}' | base64 -d

# 端口转发访问 Grafana
kubectl port-forward svc/grafana 3000:3000 -n maas-router
```

### 3. 配置告警规则

```yaml
# prometheus-rules.yaml
groups:
  - name: maas-router-alerts
    rules:
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m]))
          / sum(rate(http_requests_total[5m])) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is above 1% for the last 5 minutes"
      
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "P99 latency is above 5 seconds"
      
      - alert: LowBalance
        expr: user_balance_cred < 10
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "User balance low"
          description: "User {{ $labels.user_id }} balance is below 10 CRED"
      
      - alert: ProviderDown
        expr: provider_health_status == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Provider {{ $labels.provider }} is down"
          description: "Provider has been down for more than 2 minutes"
      
      - alert: JudgeAgentHighLatency
        expr: judge_agent_request_duration_seconds > 0.5
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "Judge Agent high latency"
          description: "Judge Agent latency is above 500ms"
      
      - alert: DatabaseConnectionHigh
        expr: pg_stat_activity_count > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High database connection count"
          description: "Database has more than 80 active connections"
      
      - alert: PodCrashLooping
        expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Pod is crash looping"
          description: "Pod {{ $labels.pod }} is restarting frequently"
      
      - alert: HighMemoryUsage
        expr: |
          container_memory_usage_bytes{container!=""}
          / container_spec_memory_limit_bytes{container!=""} > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Container {{ $labels.container }} is using more than 90% memory"
```

### 4. 配置 Alertmanager

```yaml
# alertmanager-config.yaml
global:
  smtp_smarthost: 'smtp.gmail.com:587'
  smtp_from: 'alerts@maas-router.com'
  smtp_auth_username: 'alerts@maas-router.com'
  smtp_auth_password: 'your-email-password'

route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty'
      continue: true
    - match:
        severity: warning
      receiver: 'slack'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@maas-router.com'
        subject: 'MaaS-Router Alert: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Severity: {{ .Labels.severity }}
          Time: {{ .StartsAt }}
          {{ end }}

  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#alerts'
        title: 'MaaS-Router Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'your-pagerduty-service-key'
        severity: critical
```

### 5. 部署 Loki（日志聚合）

```bash
# 添加 Loki Helm 仓库
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# 安装 Loki
helm install loki grafana/loki-stack \
  --namespace maas-router \
  --set promtail.enabled=true
```

## 故障排查

### 1. 查看 Pod 状态

```bash
# 查看 Pod 状态
kubectl get pods -n maas-router

# 查看 Pod 详情
kubectl describe pod <pod-name> -n maas-router

# 查看 Pod 日志
kubectl logs <pod-name> -n maas-router

# 查看之前的容器日志
kubectl logs <pod-name> -n maas-router --previous
```

### 2. 网络诊断

```bash
# 进入 Pod 调试
kubectl exec -it <pod-name> -n maas-router -- /bin/sh

# 测试网络连通性
kubectl run -it --rm debug --image=busybox:1.36 --restart=Never -- nslookup postgres

# 查看 Service 端点
kubectl get endpoints -n maas-router
```

### 3. 资源使用

```bash
# 查看资源使用
kubectl top pods -n maas-router
kubectl top nodes

# 查看资源限制
kubectl describe node <node-name>
```

### 4. 事件查看

```bash
# 查看命名空间事件
kubectl get events -n maas-router --sort-by='.lastTimestamp'

# 查看警告事件
kubectl get events -n maas-router --field-selector type=Warning
```

### 5. 调试命令

```bash
# 复制文件到 Pod
kubectl cp ./local-file <pod-name>:/app/ -n maas-router

# 从 Pod 复制文件
kubectl cp <pod-name>:/app/logs/app.log ./local.log -n maas-router

# 端口转发
kubectl port-forward svc/api-gateway 8080:80 -n maas-router
```

## 生产环境检查清单

- [ ] 使用外部托管数据库（如 AWS RDS、阿里云 RDS）
- [ ] 使用外部托管 Redis（如 AWS ElastiCache、阿里云 Redis）
- [ ] 配置 Pod 安全策略
- [ ] 启用网络策略（NetworkPolicy）
- [ ] 配置资源限制（ResourceQuota、LimitRange）
- [ ] 启用审计日志
- [ ] 配置备份策略
- [ ] 配置多可用区部署
- [ ] 配置 SSL/TLS 证书自动续期
- [ ] 配置监控告警
