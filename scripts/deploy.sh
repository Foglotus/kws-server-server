#!/bin/bash

##############################################################################
# AI Recorder 离线部署脚本
# 用途：在目标机器上部署 AI Recorder 服务
# 前提：已通过 prepare.sh 准备好镜像和模型文件
##############################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 配置参数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$SCRIPT_DIR"
MODELS_DIR="$INSTALL_DIR/models"
LOGS_DIR="$INSTALL_DIR/logs"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
CONTAINER_NAME="airecorder"
IMAGE_NAME="airecorder:latest"
SERVICE_PORT="11123"

# 读取版本信息
VERSION=$(cat "$SCRIPT_DIR/VERSION" 2>/dev/null || echo "unknown")

# 部署文件
IMAGE_FILE="$SCRIPT_DIR/airecorder.tar.gz"
MODELS_FILE="$SCRIPT_DIR/models.tar.gz"
CONFIG_TEMPLATE="$SCRIPT_DIR/config.yaml"

echo "=========================================="
echo "  AI Recorder 离线部署脚本"
echo "  版本: $VERSION"
echo "=========================================="
echo ""

# 检查是否为 root 用户
# 由于现在部署在当前目录，不再需要 root 权限
# if [ "$EUID" -ne 0 ]; then 
#     log_warning "建议使用 root 权限运行此脚本"
#     log_info "可以使用: sudo ./deploy.sh"
#     echo ""
#     read -p "是否继续以当前用户部署？(y/n) " -n 1 -r
#     echo
#     if [[ ! $REPLY =~ ^[Yy]$ ]]; then
#         exit 1
#     fi
# fi

# 步骤 1: 检查部署文件
echo ""
log_info "步骤 1/5: 检查部署文件..."
echo "----------------------------------------"

if [ ! -f "$IMAGE_FILE" ]; then
    log_error "镜像文件不存在: $IMAGE_FILE"
    exit 1
fi
if [ ! -f "$MODELS_FILE" ]; then
    log_error "模型文件不存在: $MODELS_FILE"
    exit 1
fi
log_success "部署文件检查完成"

# 步骤 2: 创建目录结构
echo ""
log_info "步骤 2/5: 创建目录结构..."
echo "----------------------------------------"

log_info "创建安装目录: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$MODELS_DIR"
mkdir -p "$LOGS_DIR"

log_success "目录创建完成"

# 步骤 3: 解压模型文件
echo ""
log_info "步骤 3/5: 解压模型文件..."
echo "----------------------------------------"

log_info "解压到: $MODELS_DIR (这可能需要几分钟)"
tar -xzf "$MODELS_FILE" -C "$INSTALL_DIR" 2>/dev/null || {
    log_error "模型文件解压失败"
    exit 1
}

# 验证模型文件
REQUIRED_MODELS=(
    "$MODELS_DIR/vad/silero_vad.onnx"
    "$MODELS_DIR/streaming/sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20"
    "$MODELS_DIR/offline/sherpa-onnx-paraformer-zh-2023-09-14"
    "$MODELS_DIR/diarization/sherpa-onnx-pyannote-segmentation-3-0"
)

for model in "${REQUIRED_MODELS[@]}"; do
    if [ ! -e "$model" ]; then
        log_error "模型文件缺失: $model"
        exit 1
    fi
done

log_success "模型文件解压完成"

# 步骤 4: 加载 Docker 镜像
echo ""
log_info "步骤 4/5: 加载 Docker 镜像..."
echo "----------------------------------------"

log_info "加载镜像: $IMAGE_FILE (这可能需要几分钟)"
docker load -i "$IMAGE_FILE" 2>/dev/null || {
    log_error "Docker 镜像加载失败"
    exit 1
}

# 验证镜像
if ! docker images | grep -q "airecorder"; then
    log_error "Docker 镜像加载失败，未找到 airecorder 镜像"
    exit 1
fi

log_success "Docker 镜像加载完成"

# 步骤 5: 启动服务
echo ""
log_info "步骤 5/5: 启动服务..."
echo "----------------------------------------"

# 停止并删除旧容器（如果存在）
if docker ps -a | grep -q "$CONTAINER_NAME"; then
    log_info "检测到已存在的容器，正在停止..."
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
    log_success "旧容器已清理"
fi

# 启动新容器
log_info "启动 Docker 容器..."
docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$SERVICE_PORT:$SERVICE_PORT" \
    -v "$MODELS_DIR/streaming:/models/streaming:ro" \
    -v "$MODELS_DIR/offline:/models/offline:ro" \
    -v "$MODELS_DIR/diarization:/models/diarization:ro" \
    -v "$MODELS_DIR/vad:/models/vad:ro" \
    -v "$MODELS_DIR/punctuation:/models/punctuation:ro" \
    -v "$LOGS_DIR:/logs" \
    -e TZ=Asia/Shanghai \
    "$IMAGE_NAME" 2>/dev/null || {
    log_error "容器启动失败"
    exit 1
}

log_success "容器已启动"

# 等待服务就绪
log_info "等待服务启动..."
sleep 5

# 检查容器状态
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    log_error "容器启动失败"
    echo ""
    echo "查看日志:"
    docker logs "$CONTAINER_NAME"
    exit 1
fi

# 健康检查
log_info "执行健康检查..."
for i in {1..10}; do
    if curl -s http://localhost:$SERVICE_PORT/realkws/health > /dev/null 2>&1; then
        log_success "服务健康检查通过"
        
        # 显示版本信息
        log_info "获取服务版本信息..."
        VERSION_INFO=$(curl -s http://localhost:$SERVICE_PORT/realkws/health 2>/dev/null)
        if [ -n "$VERSION_INFO" ]; then
            echo "$VERSION_INFO" | grep -o '"version":"[^"]*"' || true
        fi
        break
    fi
    if [ $i -eq 10 ]; then
        log_warning "健康检查超时，服务可能需要更长时间启动"
        log_info "可以稍后手动检查: curl http://localhost:$SERVICE_PORT/realkws/health"
    fi
    sleep 3
done

# 完成
echo ""
echo "=========================================="
log_success "AI Recorder 部署完成！"
echo "=========================================="
echo ""
echo "📋 部署信息:"
echo "  - 服务名称: $CONTAINER_NAME"
echo "  - 服务版本: $VERSION"
echo "  - 镜像版本: $IMAGE_NAME"
echo "  - 安装目录: $INSTALL_DIR"
echo "  - 模型目录: $MODELS_DIR"
echo "  - 日志目录: $LOGS_DIR"
echo "  - 配置文件: $CONFIG_FILE"
echo ""
echo "🌐 访问地址:"
echo "  - HTTP 服务: http://localhost:$SERVICE_PORT"
echo "  - Web 界面: http://localhost:$SERVICE_PORT"
echo "  - 健康检查: http://localhost:$SERVICE_PORT/realkws/health"
echo ""
echo "🔧 管理命令:"
echo "  - 查看状态: docker ps | grep $CONTAINER_NAME"
echo "  - 查看日志: docker logs -f $CONTAINER_NAME"
echo "  - 查看版本: docker exec $CONTAINER_NAME ./airecorder -v"
echo "  - 停止服务: docker stop $CONTAINER_NAME"
echo "  - 启动服务: docker start $CONTAINER_NAME"
echo "  - 重启服务: docker restart $CONTAINER_NAME"
echo ""

# 显示容器信息
echo "📊 容器状态:"
docker ps | grep "$CONTAINER_NAME" || true
echo ""
