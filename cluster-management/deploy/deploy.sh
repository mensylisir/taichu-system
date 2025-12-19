#!/bin/bash

# 太初集群管理系统 Kubernetes 部署脚本
# 作者: DevOps Team
# 版本: v1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖工具..."

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装，请先安装 kubectl"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "docker 未安装，请先安装 docker"
        exit 1
    fi

    log_info "依赖检查完成"
}

# 创建命名空间
create_namespace() {
    log_info "创建命名空间 taichu-system..."

    kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: taichu-system
  labels:
    name: taichu-system
    environment: production
EOF

    log_info "命名空间创建完成"
}

# 构建镜像
build_image() {
    log_info "构建 Docker 镜像..."

    # 检查是否在正确的目录
    if [ ! -f "Dockerfile" ]; then
        log_error "未找到 Dockerfile，请在项目根目录执行此脚本"
        exit 1
    fi

    # 构建镜像
    docker build -t taichu/cluster-management:latest .

    log_info "镜像构建完成"
}

# 部署到 Kubernetes
deploy_to_k8s() {
    log_info "部署到 Kubernetes..."

    # 生成随机密码
    DB_PASSWORD=$(openssl rand -base64 32)
    JWT_SECRET=$(openssl rand -base64 32)
    ENCRYPTION_KEY=$(openssl rand -base64 32)

    log_info "更新 Secret 配置..."

    # 更新 secret.yaml 中的密码
    cat <<EOF > temp_secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: taichu-cluster-management-secret
  namespace: taichu-system
  labels:
    app: taichu-cluster-management
    version: v1.0.0
type: Opaque
stringData:
  database-password: $(echo -n "$DB_PASSWORD" | base64)
  jwt-secret: $(echo -n "$JWT_SECRET" | base64)
  encryption-key: $(echo -n "$ENCRYPTION_KEY" | base64)
  admin-username: YWRtaW4=
  admin-password: YWRtaW4=
---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: taichu-system
type: Opaque
stringData:
  postgres-password: $(echo -n "$DB_PASSWORD" | base64)
  postgres-username: cG9zdGdyZXM=
EOF

    # 应用配置
    kubectl apply -f configmap.yaml
    kubectl apply -f temp_secret.yaml
    kubectl apply -f postgres.yaml
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    kubectl apply -f ingress.yaml
    kubectl apply -f hpa.yaml
    kubectl apply -f pdb.yaml
    kubectl apply -f networkpolicy.yaml

    # 清理临时文件
    rm -f temp_secret.yaml

    log_info "Kubernetes 部署完成"
}

# 等待部署完成
wait_for_deployment() {
    log_info "等待部署完成..."

    kubectl wait --for=condition=available --timeout=300s deployment/taichu-cluster-management -n taichu-system
    kubectl wait --for=condition=available --timeout=300s statefulset/postgres -n taichu-system

    log_info "部署完成"
}

# 显示部署状态
show_status() {
    log_info "显示部署状态..."

    echo ""
    echo "=== Pod 状态 ==="
    kubectl get pods -n taichu-system

    echo ""
    echo "=== Service 状态 ==="
    kubectl get services -n taichu-system

    echo ""
    echo "=== Ingress 状态 ==="
    kubectl get ingress -n taichu-system

    echo ""
    echo "=== HPA 状态 ==="
    kubectl get hpa -n taichu-system

    echo ""
    log_info "API 服务地址: https://api.taichu.example.com"
    log_info "默认管理员账号: admin/admin"
}

# 清理函数
cleanup() {
    if [ -f "temp_secret.yaml" ]; then
        rm -f temp_secret.yaml
    fi
}

# 主函数
main() {
    echo "======================================"
    echo "  太初集群管理系统 Kubernetes 部署"
    echo "======================================"
    echo ""

    # 检查参数
    if [ "$1" == "--skip-build" ]; then
        log_warn "跳过镜像构建步骤"
        SKIP_BUILD=true
    fi

    # 设置陷阱
    trap cleanup EXIT

    # 执行部署步骤
    check_dependencies

    if [ "$SKIP_BUILD" != true ]; then
        build_image
    fi

    create_namespace
    deploy_to_k8s
    wait_for_deployment
    show_status

    echo ""
    log_info "部署完成！"
}

# 显示帮助信息
if [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --skip-build    跳过镜像构建步骤"
    echo "  -h, --help      显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    # 完整部署"
    echo "  $0 --skip-build       # 跳过镜像构建"
    exit 0
fi

# 执行主函数
main "$@"
