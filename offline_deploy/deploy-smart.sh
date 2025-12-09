#!/bin/bash

##############################################################################
# AI Recorder 智能部署脚本
# 支持两种模式：
#   1. 完整部署模式 - 包含 Docker 镜像
#   2. 快速更新模式 - 仅包含编译好的程序
#
# 使用方法：
#   ./deploy.sh           # 交互式部署
#   ./deploy.sh --yes     # 自动部署（跳过确认）
##############################################################################

set -e

# 解析参数
AUTO_YES=false
for arg in "$@"; do
    case $arg in
        -y|--yes)
            AUTO_YES=true
            shift
            ;;
    esac
done

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 打印函数
log_info() { printf "${BLUE}[INFO]${NC} %s\n" "$1"; }
log_success() { printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"; }
log_error() { printf "${RED}[ERROR]${NC} %s\n" "$1"; }
log_warning() { printf "${YELLOW}[WARNING]${NC} %s\n" "$1"; }

# 配置参数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$SCRIPT_DIR"
VERSION=$(cat "$SCRIPT_DIR/VERSION" 2>/dev/null || echo "unknown")

# 部署文件检测
IMAGE_FILE="$SCRIPT_DIR/airecorder.tar.gz"
BASE_IMAGE_FILE="$SCRIPT_DIR/airecorder-base.tar.gz"
MODELS_FILE="$SCRIPT_DIR/models.tar.gz"
BINARY_FILE="$SCRIPT_DIR/bin/airecorder"
DOCKER_COMPOSE_RUNTIME="$SCRIPT_DIR/docker-compose.runtime.yml"
DOCKER_COMPOSE_STANDARD="$SCRIPT_DIR/docker-compose.yml"

echo "=========================================="
echo "  AI Recorder 智能部署脚本"
echo "  版本: $VERSION"
echo "=========================================="
echo ""
echo "提示: 使用 ./deploy.sh --yes 可跳过确认直接部署"
echo ""

# 自动检测部署模式
detect_deployment_mode() {
    if [ -f "$BINARY_FILE" ] && [ -f "$BASE_IMAGE_FILE" ]; then
        echo "runtime"
    elif [ -f "$IMAGE_FILE" ]; then
        echo "standard"
    else
        echo "unknown"
    fi
}

DEPLOY_MODE=$(detect_deployment_mode)

case $DEPLOY_MODE in
    runtime)
        log_info "检测到【运行时部署模式】- 基础镜像 + 编译程序"
        log_info "  包含: 基础镜像 + 编译好的程序"
        log_info "  优势: 快速更新，仅传输程序文件"
        ;;
    standard)
        log_info "检测到【标准部署模式】- 完整 Docker 镜像"
        log_info "  包含: 完整 Docker 镜像（包含程序）"
        log_info "  优势: 完整部署，一次性配置"
        ;;
    unknown)
        log_error "未检测到有效的部署文件！"
        echo ""
        echo "请确保以下文件之一存在："
        echo "  1. 运行时模式: bin/airecorder + airecorder-base.tar.gz"
        echo "  2. 标准模式: airecorder.tar.gz"
        exit 1
        ;;
esac

echo ""
if [ "$AUTO_YES" = false ]; then
    read -p "是否继续部署？(y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
else
    log_info "自动部署模式，跳过确认"
fi

# 检查 Docker
echo ""
log_info "检查 Docker 环境..."
if ! command -v docker &> /dev/null; then
    log_error "Docker 未安装！"
    echo ""
    echo "请先安装 Docker:"
    echo "  Ubuntu/Debian: curl -fsSL https://get.docker.com | sh"
    echo "  CentOS/RHEL: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

if ! docker ps &> /dev/null; then
    log_error "Docker 未运行或权限不足！"
    echo ""
    echo "请确保:"
    echo "  1. Docker 服务已启动: sudo systemctl start docker"
    echo "  2. 当前用户在 docker 组: sudo usermod -aG docker $USER"
    exit 1
fi
log_success "Docker 环境正常"

# 创建目录
echo ""
log_info "创建必要目录..."
mkdir -p "$INSTALL_DIR/models"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/bin/lib"
mkdir -p "$INSTALL_DIR/static"
log_success "目录创建完成"

# 解压模型文件
if [ -f "$MODELS_FILE" ]; then
    echo ""
    log_info "解压模型文件..."
    log_info "目标: $INSTALL_DIR/models (可能需要几分钟)"
    
    tar -xzf "$MODELS_FILE" -C "$INSTALL_DIR" || {
        log_error "模型文件解压失败"
        exit 1
    }
    
    log_success "模型文件解压完成"
    
    # 验证关键模型
    log_info "验证模型文件..."
    REQUIRED_MODELS=(
        "models/vad/silero_vad.onnx"
    )
    
    for model in "${REQUIRED_MODELS[@]}"; do
        if [ ! -e "$INSTALL_DIR/$model" ]; then
            log_error "缺少必要的模型文件: $model"
            exit 1
        fi
    done
    log_success "模型验证通过"
else
    log_warning "未找到模型文件，跳过模型解压"
    log_info "如需模型，请单独传输 models/ 目录"
fi

# 根据模式进行部署
echo ""
if [ "$DEPLOY_MODE" = "runtime" ]; then
    ##########################################################################
    # 运行时部署模式
    ##########################################################################
    log_info "执行运行时部署..."
    echo "----------------------------------------"
    
    # 停止旧服务
    if docker ps -a | grep -q "$INSTALL_DIR"; then
        log_info "停止现有服务..."
        docker-compose -f "$DOCKER_COMPOSE_RUNTIME" down 2>/dev/null || true
    fi
    
    # 加载基础镜像
    if docker images | grep -q "airecorder-base"; then
        log_info "基础镜像已存在，跳过加载"
    else
        log_info "加载基础镜像..."
        docker load -i "$BASE_IMAGE_FILE" || {
            log_error "基础镜像加载失败"
            exit 1
        }
        log_success "基础镜像加载完成"
    fi
    
    # 验证程序文件
    if [ ! -f "$BINARY_FILE" ]; then
        log_error "程序文件不存在: $BINARY_FILE"
        exit 1
    fi
    
    if [ ! -x "$BINARY_FILE" ]; then
        log_info "设置程序执行权限..."
        chmod +x "$BINARY_FILE"
    fi
    
    # 复制配置文件（避免自己复制自己）
    if [ -f "$SCRIPT_DIR/config.yaml" ] && [ "$SCRIPT_DIR" != "$INSTALL_DIR" ]; then
        cp "$SCRIPT_DIR/config.yaml" "$INSTALL_DIR/"
    fi
    
    # 复制静态文件（避免自己复制自己）
    if [ -d "$SCRIPT_DIR/static" ] && [ "$SCRIPT_DIR" != "$INSTALL_DIR" ]; then
        cp -r "$SCRIPT_DIR/static"/* "$INSTALL_DIR/static/" 2>/dev/null || true
    fi
    
    # 创建或使用 docker-compose 配置
    if [ ! -f "$DOCKER_COMPOSE_RUNTIME" ]; then
        log_info "创建 docker-compose 配置..."
        cat > "$DOCKER_COMPOSE_RUNTIME" << 'EOF'
version: "3.8"

services:
  airecorder:
    image: airecorder-base:latest
    container_name: airecorder
    restart: unless-stopped
    ports:
      - "11123:11123"
    volumes:
      - ./bin/airecorder:/app/airecorder:ro
      - ./bin/lib:/usr/local/lib:ro
      - ./static:/app/static:ro
      - ./models/streaming:/models/streaming:ro
      - ./models/offline:/models/offline:ro
      - ./models/diarization:/models/diarization:ro
      - ./models/vad:/models/vad:ro
      - ./models/punctuation:/models/punctuation:ro
      - ./config.yaml:/app/config.yaml:ro
      - ./logs:/logs
    environment:
      - CONFIG_PATH=/app/config.yaml
      - TZ=Asia/Shanghai
      - LD_LIBRARY_PATH=/usr/local/lib
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--method=GET", "-O", "/dev/null", "http://localhost:11123/realkws/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
EOF
    fi
    
    # 启动服务
    log_info "启动服务..."
    docker-compose -f "$DOCKER_COMPOSE_RUNTIME" up -d
    
else
    ##########################################################################
    # 标准部署模式
    ##########################################################################
    log_info "执行标准部署..."
    echo "----------------------------------------"
    
    # 停止旧服务
    if docker ps -a | grep -q "airecorder"; then
        log_info "停止现有服务..."
        docker-compose -f "$DOCKER_COMPOSE_STANDARD" down 2>/dev/null || \
        docker stop airecorder 2>/dev/null || true
        docker rm airecorder 2>/dev/null || true
    fi
    
    # 加载镜像
    log_info "加载 Docker 镜像 (可能需要几分钟)..."
    docker load -i "$IMAGE_FILE" || {
        log_error "镜像加载失败"
        exit 1
    }
    log_success "镜像加载完成"
    
    # 复制配置文件（避免自己复制自己）
    if [ -f "$SCRIPT_DIR/config.yaml" ] && [ "$SCRIPT_DIR" != "$INSTALL_DIR" ]; then
        cp "$SCRIPT_DIR/config.yaml" "$INSTALL_DIR/"
    fi
    
    # 创建或使用 docker-compose 配置
    if [ ! -f "$DOCKER_COMPOSE_STANDARD" ]; then
        log_info "创建 docker-compose 配置..."
        cat > "$DOCKER_COMPOSE_STANDARD" << 'EOF'
version: "3.8"

services:
  airecorder:
    image: airecorder:latest
    container_name: airecorder
    restart: unless-stopped
    ports:
      - "11123:11123"
    volumes:
      - ./models/streaming:/models/streaming:ro
      - ./models/offline:/models/offline:ro
      - ./models/diarization:/models/diarization:ro
      - ./models/vad:/models/vad:ro
      - ./models/punctuation:/models/punctuation:ro
      - ./config.yaml:/app/config.yaml:ro
      - ./logs:/logs
    environment:
      - CONFIG_PATH=/app/config.yaml
      - TZ=Asia/Shanghai
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--method=GET", "-O", "/dev/null", "http://localhost:11123/realkws/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
EOF
    fi
    
    # 启动服务
    log_info "启动服务..."
    docker-compose -f "$DOCKER_COMPOSE_STANDARD" up -d
fi

# 等待服务启动
echo ""
log_info "等待服务启动..."
sleep 10

# 检查服务状态
echo ""
log_info "检查服务状态..."
if [ "$DEPLOY_MODE" = "runtime" ]; then
    docker-compose -f "$DOCKER_COMPOSE_RUNTIME" ps
else
    docker-compose -f "$DOCKER_COMPOSE_STANDARD" ps
fi

# 健康检查
echo ""
log_info "健康检查..."
MAX_RETRIES=12
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:11123/realkws/health | grep -q "healthy"; then
        log_success "服务运行正常！"
        echo ""
        echo "=========================================="
        echo "  ✓ 部署成功！"
        echo "=========================================="
        echo ""
        echo "服务信息:"
        echo "  - 地址: http://localhost:11123"
        echo "  - 版本: $VERSION"
        echo "  - 模式: $DEPLOY_MODE"
        echo ""
        echo "常用命令:"
        if [ "$DEPLOY_MODE" = "runtime" ]; then
            echo "  - 查看日志: docker-compose -f docker-compose.runtime.yml logs -f"
            echo "  - 重启服务: docker-compose -f docker-compose.runtime.yml restart"
            echo "  - 停止服务: docker-compose -f docker-compose.runtime.yml down"
            echo "  - 查看状态: docker-compose -f docker-compose.runtime.yml ps"
        else
            echo "  - 查看日志: docker-compose logs -f"
            echo "  - 重启服务: docker-compose restart"
            echo "  - 停止服务: docker-compose down"
            echo "  - 查看状态: docker-compose ps"
        fi
        echo ""
        echo "测试接口:"
        echo "  curl http://localhost:11123/realkws/health"
        echo "  curl http://localhost:11123/"
        echo ""
        exit 0
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    log_info "等待服务就绪... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 5
done

log_error "服务启动超时或异常！"
echo ""
echo "请检查日志:"
if [ "$DEPLOY_MODE" = "runtime" ]; then
    echo "  docker-compose -f docker-compose.runtime.yml logs"
else
    echo "  docker-compose logs"
fi
exit 1
