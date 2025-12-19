#!/bin/bash

# 太初集群管理系统 Kubernetes 卸载脚本
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

# 确认卸载
confirm_uninstall() {
    log_warn "此操作将删除 taichu-system 命名空间下的所有资源！"
    echo ""
    read -p "确认卸载吗？(yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        log_info "卸载已取消"
        exit 0
    fi
}

# 删除部署资源
delete_deployments() {
    log_info "删除部署资源..."

    kubectl delete -f hpa.yaml --ignore-not-found=true
    kubectl delete -f pdb.yaml --ignore-not-found=true
    kubectl delete -f networkpolicy.yaml --ignore-not-found=true
    kubectl delete -f ingress.yaml --ignore-not-found=true
    kubectl delete -f service.yaml --ignore-not-found=true
    kubectl delete -f deployment.yaml --ignore-not-found=true
    kubectl delete -f postgres.yaml --ignore-not-found=true
    kubectl delete -f configmap.yaml --ignore-not-found=true
    kubectl delete -f secret.yaml --ignore-not-found=true

    log_info "部署资源删除完成"
}

# 等待资源删除
wait_for_deletion() {
    log_info "等待资源删除..."

    # 等待 Deployment 删除
    kubectl wait --for=delete deployment/taichu-cluster-management -n taichu-system --timeout=300s 2>/dev/null || true
    kubectl wait --for=delete statefulset/postgres -n taichu-system --timeout=300s 2>/dev/null || true

    log_info "资源删除完成"
}

# 删除命名空间
delete_namespace() {
    log_info "删除命名空间..."

    kubectl delete namespace taichu-system --ignore-not-found=true

    log_info "命名空间删除完成"
}

# 清理镜像
cleanup_images() {
    log_info "清理 Docker 镜像..."

    read -p "是否删除本地构建的镜像？(yes/no): " confirm
    if [ "$confirm" == "yes" ]; then
        docker rmi taichu/cluster-management:latest --ignore-not-found=true || true
        log_info "镜像删除完成"
    else
        log_info "跳过镜像删除"
    fi
}

# 显示状态
show_status() {
    log_info "检查剩余资源..."

    echo ""
    echo "=== 命名空间状态 ==="
    kubectl get namespaces | grep taichu || echo "没有找到 taichu 相关命名空间"

    echo ""
    echo "=== PVC 状态 ==="
    kubectl get pvc -n taichu-system 2>/dev/null || echo "没有找到相关 PVC"

    log_info "卸载完成"
}

# 主函数
main() {
    echo "======================================"
    echo "  太初集群管理系统 Kubernetes 卸载"
    echo "======================================"
    echo ""

    # 确认卸载
    confirm_uninstall

    # 卸载步骤
    delete_deployments
    wait_for_deletion
    delete_namespace
    show_status

    # 清理镜像
    cleanup_images

    echo ""
    log_info "卸载完成！"
}

# 显示帮助信息
if [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
    echo "用法: $0 [选项]"
    echo ""
    echo "此脚本将删除所有与太初集群管理系统相关的 Kubernetes 资源"
    echo ""
    echo "示例:"
    echo "  $0    # 开始卸载"
    echo ""
    exit 0
fi

# 执行主函数
main "$@"
