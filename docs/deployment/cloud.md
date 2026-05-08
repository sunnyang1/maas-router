# MaaS-Router 云端部署指南

本文档介绍如何在主流云平台上部署 MaaS-Router 项目，包括 AWS、阿里云和腾讯云。

## 目录

- [AWS 部署](#aws-部署)
- [阿里云部署](#阿里云部署)
- [腾讯云部署](#腾讯云部署)
- [一键部署脚本](#一键部署脚本)

## AWS 部署

### 方案一：ECS (Elastic Container Service)

#### 1. 创建 ECS 集群

```bash
# 创建 ECS 集群
aws ecs create-cluster --cluster-name maas-router-cluster

# 创建 VPC 和子网
aws ec2 create-vpc --cidr-block 10.0.0.0/16
aws ec2 create-subnet --vpc-id vpc-xxxx --cidr-block 10.0.1.0/24 --availability-zone us-east-1a
aws ec2 create-subnet --vpc-id vpc-xxxx --cidr-block 10.0.2.0/24 --availability-zone us-east-1b
```

#### 2. 创建 RDS PostgreSQL

```bash
# 创建 RDS 子网组
aws rds create-db-subnet-group \
  --db-subnet-group-name maas-router-db-subnet \
  --db-subnet-group-description "MaaS Router DB Subnet" \
  --subnet-ids '["subnet-xxxx","subnet-yyyy"]'

# 创建 RDS 实例
aws rds create-db-instance \
  --db-instance-identifier maas-router-db \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 16.1 \
  --allocated-storage 100 \
  --master-username maas_user \
  --master-user-password 'SecurePassword123!' \
  --vpc-security-group-ids sg-xxxx \
  --db-subnet-group-name maas-router-db-subnet \
  --backup-retention-period 7 \
  --preferred-backup-window 03:00-04:00 \
  --enable-performance-insights \
  --performance-insights-retention-period 7
```

#### 3. 创建 ElastiCache Redis

```bash
# 创建 Redis 子网组
aws elasticache create-cache-subnet-group \
  --cache-subnet-group-name maas-router-redis \
  --cache-subnet-group-description "MaaS Router Redis" \
  --subnet-ids 'subnet-xxxx' 'subnet-yyyy'

# 创建 Redis 集群
aws elasticache create-replication-group \
  --replication-group-id maas-router-redis \
  --replication-group-description "MaaS Router Redis Cluster" \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-clusters 2 \
  --automatic-failover-enabled \
  --multi-az-enabled \
  --cache-subnet-group-name maas-router-redis \
  --security-group-ids sg-xxxx
```

#### 4. 创建 ECS 任务定义

```json
{
  "family": "maas-router-backend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::123456789:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::123456789:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "123456789.dkr.ecr.us-east-1.amazonaws.com/maas-router/backend:v1.0.0",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "MAAS_ROUTER_DATABASE_HOST",
          "value": "maas-router-db.xxxx.us-east-1.rds.amazonaws.com"
        },
        {
          "name": "MAAS_ROUTER_DATABASE_PORT",
          "value": "5432"
        },
        {
          "name": "MAAS_ROUTER_REDIS_HOST",
          "value": "maas-router-redis.xxxx.cache.amazonaws.com"
        }
      ],
      "secrets": [
        {
          "name": "MAAS_ROUTER_DATABASE_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:maas-router/db-password"
        },
        {
          "name": "MAAS_ROUTER_JWT_SECRET",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:maas-router/jwt-secret"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/maas-router",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "backend"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

#### 5. 创建 ECS 服务

```bash
# 创建 Application Load Balancer
aws elbv2 create-load-balancer \
  --name maas-router-alb \
  --subnets subnet-xxxx subnet-yyyy \
  --security-groups sg-xxxx \
  --scheme internet-facing \
  --type application

# 创建目标组
aws elbv2 create-target-group \
  --name maas-router-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-xxxx \
  --target-type ip \
  --health-check-path /health

# 创建 ECS 服务
aws ecs create-service \
  --cluster maas-router-cluster \
  --service-name maas-router-backend \
  --task-definition maas-router-backend:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxxx,subnet-yyyy],securityGroups=[sg-xxxx],assignPublicIp=ENABLED}" \
  --load-balancers targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/maas-router-tg/xxxx,containerName=backend,containerPort=8080
```

### 方案二：EKS (Elastic Kubernetes Service)

#### 1. 创建 EKS 集群

```bash
# 使用 eksctl 创建集群
eksctl create cluster \
  --name maas-router-cluster \
  --region us-east-1 \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 10 \
  --managed

# 配置 kubectl
aws eks update-kubeconfig --region us-east-1 --name maas-router-cluster
```

#### 2. 安装 AWS Load Balancer Controller

```bash
# 下载 IAM 策略
curl -O https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/v2.6.2/docs/install/iam_policy.json

# 创建 IAM 策略
aws iam create-policy \
  --policy-name AWSLoadBalancerControllerIAMPolicy \
  --policy-document file://iam_policy.json

# 创建 Service Account
eksctl create iamserviceaccount \
  --cluster=maas-router-cluster \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --attach-policy-arn=arn:aws:iam::123456789:policy/AWSLoadBalancerControllerIAMPolicy \
  --approve

# 安装 AWS Load Balancer Controller
helm repo add eks https://aws.github.io/eks-charts
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=maas-router-cluster \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller
```

#### 3. 创建 Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: maas-router
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-east-1:123456789:certificate/xxxx
    alb.ingress.kubernetes.io/healthcheck-path: /health
spec:
  ingressClassName: alb
  rules:
    - host: api.maas-router.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-gateway
                port:
                  number: 80
```

## 阿里云部署

### 方案一：ACK (容器服务 Kubernetes 版)

#### 1. 创建 ACK 集群

```bash
# 使用 aliyun CLI 创建集群
aliyun cs POST /clusters \
  --body '{
    "name": "maas-router-cluster",
    "cluster_type": "ManagedKubernetes",
    "region_id": "cn-hangzhou",
    "vpc_id": "vpc-xxxx",
    "vswitch_ids": ["vsw-xxxx", "vsw-yyyy"],
    "worker_instance_types": ["ecs.g7.xlarge"],
    "num_of_nodes": 3,
    "worker_system_disk_category": "cloud_essd",
    "worker_system_disk_size": 100,
    "key_pair": "maas-router-key",
    "snat_entry": true,
    "endpoint_public_access": true,
    "security_group_id": "sg-xxxx"
  }'

# 配置 kubectl
aliyun cs GET /k8s/[cluster-id]/user_config | jq -r '.config' > ~/.kube/config
```

#### 2. 创建 RDS PostgreSQL

```bash
# 创建 RDS 实例
aliyun rds CreateDBInstance \
  --RegionId cn-hangzhou \
  --Engine PostgreSQL \
  --EngineVersion 16.0 \
  --DBInstanceClass rds.pg.c2.xlarge \
  --DBInstanceStorage 100 \
  --DBInstanceStorageType cloud_essd \
  --DBInstanceDescription "MaaS Router Database" \
  --SecurityIPList "10.0.0.0/8"

# 创建数据库账号
aliyun rds CreateAccount \
  --DBInstanceId rm-xxxx \
  --AccountName maas_user \
  --AccountPassword 'SecurePassword123!' \
  --AccountType Super

# 创建数据库
aliyun rds CreateDatabase \
  --DBInstanceId rm-xxxx \
  --DBName maas_router \
  --CharacterSetName UTF8
```

#### 3. 创建云数据库 Redis 版

```bash
# 创建 Redis 实例
aliyun r-kvstore CreateInstance \
  --RegionId cn-hangzhou \
  --InstanceType Redis \
  --EngineVersion 7.0 \
  --InstanceClass redis.master.large.default \
  --VpcId vpc-xxxx \
  --VSwitchId vsw-xxxx \
  --SecurityGroupId sg-xxxx \
  --InstanceName "MaaS Router Redis"
```

#### 4. 配置 Ingress（ALB）

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: maas-router
  annotations:
    alb.ingress.kubernetes.io/scheme: internet
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-ids: "cert-xxxx"
    alb.ingress.kubernetes.io/healthcheck-path: /health
    alb.ingress.kubernetes.io/healthcheck-interval-seconds: "15"
spec:
  ingressClassName: alb
  rules:
    - host: api.maas-router.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-gateway
                port:
                  number: 80
```

### 方案二：函数计算 FC

```yaml
# serverless.yaml
edition: 3.0.0
name: maas-router
access: default

vars:
  region: cn-hangzhou
  service:
    name: maas-router-service
    description: MaaS Router Service
    nasConfig: auto
    vpcConfig:
      vpcId: vpc-xxxx
      vSwitchIds:
        - vsw-xxxx
      securityGroupId: sg-xxxx

services:
  backend:
    component: fc
    props:
      region: ${vars.region}
      service: ${vars.service}
      function:
        name: maas-router-backend
        runtime: custom-container
        cpu: 2
        memorySize: 4096
        timeout: 300
        caPort: 8080
        customContainerConfig:
          image: registry.cn-hangzhou.aliyuncs.com/maas-router/backend:v1.0.0
        environmentVariables:
          MAAS_ROUTER_DATABASE_HOST: rm-xxxx.pg.rds.aliyuncs.com
          MAAS_ROUTER_DATABASE_PORT: "5432"
          MAAS_ROUTER_DATABASE_USER: maas_user
          MAAS_ROUTER_DATABASE_DATABASE: maas_router
      triggers:
        - name: http-trigger
          type: http
          config:
            authType: anonymous
            methods:
              - GET
              - POST
              - PUT
              - DELETE
```

## 腾讯云部署

### 方案一：TKE (腾讯云容器服务)

#### 1. 创建 TKE 集群

```bash
# 使用 tccli 创建集群
tccli tke CreateCluster \
  --ClusterName maas-router-cluster \
  --ClusterCIDRSettings.ClusterCIDR 10.0.0.0/16 \
  --ClusterCIDRSettings.MaxNodePodNum 64 \
  --ClusterCIDRSettings.MaxClusterServiceNum 256 \
  --ClusterBasicSettings.ClusterOs "TencentOS Server 3.1 (TK4)" \
  --ClusterBasicSettings.ClusterVersion "1.28" \
  --ClusterBasicSettings.VpcId vpc-xxxx \
  --ClusterBasicSettings.SubnetIds '["subnet-xxxx","subnet-yyyy"]' \
  --ClusterBasicSettings.ClusterDescription "MaaS Router Production Cluster" \
  --InstanceAdvancedSettings.DockerGraphPath /var/lib/docker

# 添加节点
tccli tke CreateClusterInstances \
  --ClusterId cls-xxxx \
  --RunInstancePara '{
    "InstanceType": "S5.2XLARGE8",
    "SystemDisk": {
      "DiskType": "CLOUD_SSD",
      "DiskSize": 100
    },
    "VirtualPrivateCloud": {
      "VpcId": "vpc-xxxx",
      "SubnetId": "subnet-xxxx"
    },
    "InstanceCount": 3
  }'

# 配置 kubectl
tccli tke DescribeClusterKubeconfig --ClusterId cls-xxxx | jq -r '.Kubeconfig' > ~/.kube/config
```

#### 2. 创建云数据库 PostgreSQL

```bash
# 创建 PostgreSQL 实例
tccli postgres CreateDBInstances \
  --SpecCode cdb.pg.c2.large \
  --Storage 100 \
  --InstanceChargeType POSTPAID_BY_HOUR \
  --Period 1 \
  --VpcId vpc-xxxx \
  --SubnetId subnet-xxxx \
  --DBVersion 16 \
  --InstanceName maas-router-db \
  --Zone ap-guangzhou-2 \
  --Charset UTF8 \
  --ProjectId 0

# 创建账号
tccli postgres CreateAccount \
  --DBInstanceId postgres-xxxx \
  --UserName maas_user \
  --Password 'SecurePassword123!'

# 创建数据库
tccli postgres CreateDatabase \
  --DBInstanceId postgres-xxxx \
  --DBName maas_router \
  --Owner maas_user
```

#### 3. 创建云数据库 Redis

```bash
# 创建 Redis 实例
tccli redis CreateInstances \
  --ZoneId 100002 \
  --TypeId 7 \
  --MemSize 4096 \
  --GoodsNum 1 \
  --Period 1 \
  --BillingMode 0 \
  --Password 'RedisPassword123!' \
  --VpcId vpc-xxxx \
  --SubnetId subnet-xxxx \
  --InstanceName maas-router-redis \
  --AutoRenew 1
```

#### 4. 配置 CLB Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: maas-router
  annotations:
    kubernetes.io/ingress.class: qcloud
    kubernetes.io/ingress.extensiveParameters: '{"AddressIPVersion": "IPV4"}'
    kubernetes.io/ingress.qcloud-loadbalance-id: lb-xxxx
    ingress.cloud.tencent.com/enable-grace-shutdown: "true"
    ingress.cloud.tencent.com/healthcheck-path: /health
    ingress.cloud.tencent.com/healthcheck-interval-seconds: "15"
spec:
  tls:
    - hosts:
        - api.maas-router.com
      secretName: api-gateway-tls
  rules:
    - host: api.maas-router.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-gateway
                port:
                  number: 80
```

### 方案二：Serverless 云函数 SCF

```yaml
# serverless.yml
component: scf
name: maas-router

inputs:
  name: maas-router-backend
  namespace: default
  type: web
  region: ap-guangzhou
  runtime: CustomContainer
  memorySize: 4096
  timeout: 300
  cpu: 2
  environment:
    variables:
      MAAS_ROUTER_DATABASE_HOST: postgres-xxxx.sql.tencentcdb.com
      MAAS_ROUTER_DATABASE_PORT: "5432"
      MAAS_ROUTER_DATABASE_USER: maas_user
      MAAS_ROUTER_DATABASE_DATABASE: maas_router
  customContainer:
    image: ccr.ccs.tencentyun.com/maas-router/backend:v1.0.0
    port: 8080
  vpc:
    vpcId: vpc-xxxx
    subnetId: subnet-xxxx
  events:
    - http:
        path: /
        method: ANY
```

## 一键部署脚本

### AWS 一键部署

```bash
#!/bin/bash
# deploy-aws.sh

set -e

export AWS_REGION=${AWS_REGION:-us-east-1}
export CLUSTER_NAME=${CLUSTER_NAME:-maas-router}
export DOMAIN=${DOMAIN:-api.maas-router.com}

echo "=== 开始 AWS 部署 ==="

# 1. 创建 ECR 仓库
echo "创建 ECR 仓库..."
aws ecr create-repository --repository-name maas-router/backend --region $AWS_REGION || true
aws ecr create-repository --repository-name maas-router/judge-agent --region $AWS_REGION || true

# 2. 登录 ECR
echo "登录 ECR..."
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $(aws sts get-caller-identity --query Account --output text).dkr.ecr.$AWS_REGION.amazonaws.com

# 3. 构建并推送镜像
echo "构建镜像..."
docker build -t maas-router/backend:v1.0.0 ./backend
docker tag maas-router/backend:v1.0.0 $(aws sts get-caller-identity --query Account --output text).dkr.ecr.$AWS_REGION.amazonaws.com/maas-router/backend:v1.0.0
docker push $(aws sts get-caller-identity --query Account --output text).dkr.ecr.$AWS_REGION.amazonaws.com/maas-router/backend:v1.0.0

# 4. 创建 EKS 集群
echo "创建 EKS 集群..."
eksctl create cluster \
  --name $CLUSTER_NAME \
  --region $AWS_REGION \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 10 \
  --managed \
  --asg-access \
  --external-dns-access \
  --full-ecr-access

# 5. 创建 RDS
echo "创建 RDS..."
aws rds create-db-instance \
  --db-instance-identifier ${CLUSTER_NAME}-db \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 16.1 \
  --allocated-storage 100 \
  --master-username maas_user \
  --master-user-password $(openssl rand -base64 32) \
  --backup-retention-period 7 \
  --no-cli-pager || true

# 6. 创建 ElastiCache
echo "创建 ElastiCache..."
aws elasticache create-replication-group \
  --replication-group-id ${CLUSTER_NAME}-redis \
  --replication-group-description "MaaS Router Redis" \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-clusters 2 \
  --automatic-failover-enabled \
  --no-cli-pager || true

# 7. 部署应用
echo "部署应用..."
kubectl create namespace maas-router || true
kubectl apply -f infra/k8s/secrets.yaml
kubectl apply -f infra/k8s/configmap.yaml
kubectl apply -f infra/k8s/api-gateway.yaml

# 8. 安装 Ingress Controller
echo "安装 Ingress Controller..."
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer

echo "=== 部署完成 ==="
echo "API 地址: http://$(kubectl get svc ingress-nginx-controller -n ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')"
```

### 阿里云一键部署

```bash
#!/bin/bash
# deploy-aliyun.sh

set -e

export REGION=${REGION:-cn-hangzhou}
export CLUSTER_NAME=${CLUSTER_NAME:-maas-router}
export DOMAIN=${DOMAIN:-api.maas-router.com}

echo "=== 开始阿里云部署 ==="

# 1. 创建容器镜像服务命名空间
echo "创建 ACR 命名空间..."
aliyun cr CreateNamespace --RegionId $REGION --NamespaceName maas-router || true

# 2. 登录 ACR
echo "登录 ACR..."
aliyun cr GetAuthorizationToken --RegionId $REGION | jq -r '.Data.AuthorizationToken' | docker login --username=cr_temp_user --password-stdin registry.$REGION.aliyuncs.com

# 3. 构建并推送镜像
echo "构建镜像..."
docker build -t maas-router/backend:v1.0.0 ./backend
docker tag maas-router/backend:v1.0.0 registry.$REGION.aliyuncs.com/maas-router/backend:v1.0.0
docker push registry.$REGION.aliyuncs.com/maas-router/backend:v1.0.0

# 4. 创建 ACK 集群
echo "创建 ACK 集群..."
aliyun cs POST /clusters --body "{
  \"name\": \"$CLUSTER_NAME\",
  \"cluster_type\": \"ManagedKubernetes\",
  \"region_id\": \"$REGION\",
  \"num_of_nodes\": 3,
  \"worker_instance_types\": [\"ecs.g7.xlarge\"]
}"

# 5. 创建 RDS
echo "创建 RDS..."
aliyun rds CreateDBInstance \
  --RegionId $REGION \
  --Engine PostgreSQL \
  --EngineVersion 16.0 \
  --DBInstanceClass rds.pg.c2.xlarge \
  --DBInstanceStorage 100 || true

# 6. 创建 Redis
echo "创建 Redis..."
aliyun r-kvstore CreateInstance \
  --RegionId $REGION \
  --InstanceType Redis \
  --EngineVersion 7.0 \
  --InstanceClass redis.master.large.default || true

# 7. 部署应用
echo "部署应用..."
kubectl create namespace maas-router || true
kubectl apply -f infra/k8s/

# 8. 安装 ALB Ingress Controller
echo "安装 ALB Ingress Controller..."
helm repo add aliyun https://kubernetes.oss-cn-hangzhou.aliyuncs.com/charts
helm install alb-ingress-controller aliyun/alb-ingress-controller \
  --namespace kube-system

echo "=== 部署完成 ==="
```

### 腾讯云一键部署

```bash
#!/bin/bash
# deploy-tencent.sh

set -e

export REGION=${REGION:-ap-guangzhou}
export CLUSTER_NAME=${CLUSTER_NAME:-maas-router}
export DOMAIN=${DOMAIN:-api.maas-router.com}

echo "=== 开始腾讯云部署 ==="

# 1. 创建 TCR 命名空间
echo "创建 TCR 命名空间..."
tccli tcr CreateNamespace \
  --RegistryId tcr-xxxx \
  --NamespaceName maas-router || true

# 2. 登录 TCR
echo "登录 TCR..."
docker login ccr.ccs.tencentyun.com --username=100000000001 --password=$(tccli tcr GetAuthorizationToken --RegistryId tcr-xxxx | jq -r '.Token.Password')

# 3. 构建并推送镜像
echo "构建镜像..."
docker build -t maas-router/backend:v1.0.0 ./backend
docker tag maas-router/backend:v1.0.0 ccr.ccs.tencentyun.com/maas-router/backend:v1.0.0
docker push ccr.ccs.tencentyun.com/maas-router/backend:v1.0.0

# 4. 创建 TKE 集群
echo "创建 TKE 集群..."
tccli tke CreateCluster \
  --ClusterName $CLUSTER_NAME \
  --ClusterVersion "1.28" || true

# 5. 创建 PostgreSQL
echo "创建 PostgreSQL..."
tccli postgres CreateDBInstances \
  --SpecCode cdb.pg.c2.large \
  --Storage 100 \
  --DBVersion 16 || true

# 6. 创建 Redis
echo "创建 Redis..."
tccli redis CreateInstances \
  --TypeId 7 \
  --MemSize 4096 || true

# 7. 部署应用
echo "部署应用..."
kubectl create namespace maas-router || true
kubectl apply -f infra/k8s/

# 8. 安装 CLB Ingress
echo "安装 CLB Ingress..."
kubectl apply -f https://raw.githubusercontent.com/TencentCloud/tencentcloud-cloud-controller-manager/master/docs/clb-ingress-controller.yaml

echo "=== 部署完成 ==="
```

## 成本优化建议

### AWS

1. **使用 Spot 实例**：非关键工作负载可使用 Spot 实例节省成本
2. **预留实例**：长期运行的数据库使用预留实例
3. **Savings Plans**：计算资源使用 Savings Plans
4. **自动扩缩容**：配置 HPA 和 Cluster Autoscaler

### 阿里云

1. **抢占式实例**：开发测试环境使用抢占式实例
2. **包年包月**：长期运行的数据库使用包年包月
3. **节省计划**：计算资源使用节省计划
4. **弹性伸缩**：配置自动扩缩容

### 腾讯云

1. **竞价实例**：非关键工作负载使用竞价实例
2. **包年包月**：长期资源使用包年包月
3. **预留券**：使用预留券抵扣费用
4. **弹性伸缩组**：配置自动扩缩容
