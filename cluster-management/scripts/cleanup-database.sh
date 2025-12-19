#!/bin/bash

# 数据库清理脚本
# 用途：删除并重新创建数据库，解决乱码问题

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 加载配置
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/configs/config.yaml"

if [ ! -f "$CONFIG_FILE" ]; then
    log_error "配置文件不存在: $CONFIG_FILE"
    exit 1
fi

# 解析配置文件
DB_HOST=$(grep -A 1 "host:" "$CONFIG_FILE" | grep "host:" | awk '{print $2}' | tr -d '"')
DB_PORT=$(grep -A 1 "port:" "$CONFIG_FILE" | grep "port:" | grep -v "database:" | head -1 | awk '{print $2}' | tr -d '"')
DB_USER=$(grep -A 1 "username:" "$CONFIG_FILE" | grep "username:" | awk '{print $2}' | tr -d '"')
DB_PASSWORD=$(grep -A 1 "password:" "$CONFIG_FILE" | grep "password:" | awk '{print $2}' | tr -d '"')
DB_NAME=$(grep -A 1 "dbname:" "$CONFIG_FILE" | grep "dbname:" | awk '{print $2}' | tr -d '"')

log_info "数据库配置："
log_info "  主机: $DB_HOST:$DB_PORT"
log_info "  数据库: $DB_NAME"
log_info "  用户: $DB_USER"

# 检查PostgreSQL连接
log_info "检查PostgreSQL连接..."
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "SELECT 1" > /dev/null 2>&1; then
    log_error "无法连接到PostgreSQL，请检查配置"
    exit 1
fi

# 确认操作
echo ""
log_warn "这将删除数据库 '$DB_NAME' 并重新创建"
log_warn "所有数据将丢失！"
echo ""
read -p "确认删除数据库? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    log_info "操作已取消"
    exit 0
fi

# 删除现有连接
log_info "终止现有连接..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "
    SELECT pg_terminate_backend(pid)
    FROM pg_stat_activity
    WHERE datname = '$DB_NAME';
" 2>/dev/null || true

# 删除数据库
log_info "删除数据库 '$DB_NAME'..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || {
    log_error "删除数据库失败"
    exit 1
}

# 创建数据库（UTF8编码）
log_info "创建数据库 '$DB_NAME' (UTF8编码)..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "
    CREATE DATABASE $DB_NAME
    WITH ENCODING 'UTF8'
    LC_COLLATE='zh_CN.UTF-8'
    LC_CTYPE='zh_CN.UTF-8'
    TEMPLATE=template0;
" || {
    log_error "创建数据库失败"
    exit 1
}

# 验证数据库编码
log_info "验证数据库编码..."
ENCODING=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SHOW server_encoding;" | xargs)
if [ "$ENCODING" = "UTF8" ]; then
    log_info "数据库编码设置正确: $ENCODING"
else
    log_warn "数据库编码: $ENCODING (期望: UTF8)"
fi

# 设置数据库参数
log_info "优化数据库参数..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    ALTER DATABASE $DB_NAME SET timezone TO 'Asia/Shanghai';
    ALTER DATABASE $DB_NAME SET client_encoding TO 'UTF8';
" 2>/dev/null || log_warn "部分参数设置失败，但不影响使用"

log_info "数据库清理完成！"
log_info "现在可以重新启动应用程序进行自动迁移"
echo ""
log_info "使用方法："
echo "  cd $PROJECT_ROOT"
echo "  go run ./cmd/server/main.go"
echo ""
