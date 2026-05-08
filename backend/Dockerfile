# MaaS-Router Backend Dockerfile
# 多阶段构建

# 阶段1: 构建
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 生成 Ent 代码
RUN go install entgo.io/ent/cmd/ent@latest && go generate ./ent

# 生成 Wire 代码
RUN go install github.com/google/wire/cmd/wire@latest && wire ./cmd/server

# 构建
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /maas-router ./cmd/server

# 阶段2: 运行
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /maas-router /app/
COPY --from=builder /app/configs /app/configs

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8080

# 运行
ENTRYPOINT ["/app/maas-router"]
CMD ["-config", "/app/configs/config.yaml"]
