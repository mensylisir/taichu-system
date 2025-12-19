# 多阶段构建 Dockerfile for 太初集群管理系统

# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具和依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置Go模块代理，加速依赖下载
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GOSUMDB=sum.golang.google.cn

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源码
COPY . .

# 构建应用
# 使用 CGO_ENABLED=0 确保静态编译
# 添加并发编译和缓存优化
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags '-w -s -extldflags "-static"' \
    -a -installsuffix cgo -o main cmd/server/main.go

# 第二阶段：运行阶段
FROM alpine:latest

# 安装 ca-certificates 以支持 HTTPS
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 创建日志目录
RUN mkdir -p /app/logs && \
    chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/v1/auth/token || exit 1

# 启动应用
CMD ["./main"]
