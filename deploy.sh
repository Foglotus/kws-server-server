#!/bin/bash

# AI Recorder 构建和部署脚本

set -e

# 读取版本号
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "=== AI Recorder 构建和部署 ==="
echo "版本: $VERSION"
echo "Git提交: $GIT_COMMIT"
echo "构建时间: $BUILD_TIME"
echo ""

# 解析命令行参数
TAG="latest"
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        *)
            echo "未知参数: $1"
            echo "用法: $0 [--version VERSION] [--tag TAG]"
            exit 1
            ;;
    esac
done

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose 未安装"
    exit 1
fi

# 检查模型文件是否存在
echo "检查模型文件..."
if [ ! -f "./models/vad/silero_vad.onnx" ]; then
    echo "警告: VAD 模型未找到，请先运行 ./download_models.sh"
    read -p "是否现在下载模型？ (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        bash download_models.sh
    else
        echo "跳过模型下载，构建可能失败"
    fi
fi

# 构建 Docker 镜像
echo ""
echo "构建 Docker 镜像..."
docker-compose build \
    --build-arg VERSION="$VERSION" \
    --build-arg GIT_COMMIT="$GIT_COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME"

# 打标签
if [ "$TAG" != "latest" ]; then
    echo ""
    echo "添加标签: airecorder:$TAG"
    docker tag airecorder:latest airecorder:$TAG
fi

# 启动服务
echo ""
echo "启动服务..."
docker-compose up -d

# 等待服务启动
echo ""
echo "等待服务启动..."
sleep 10

# 检查服务状态
echo ""
echo "检查服务状态..."
docker-compose ps

# 测试健康检查
echo ""
echo "测试健康检查..."
if curl -s http://localhost:11123/health | grep -q "healthy"; then
    echo "✓ 服务运行正常"
    
    # 显示版本信息
    echo ""
    echo "服务版本信息:"
    curl -s http://localhost:11123/health | python3 -m json.tool 2>/dev/null || echo "无法获取版本信息"
else
    echo "✗ 服务可能未正常启动，请检查日志"
    docker-compose logs --tail=50
    exit 1
fi

echo ""
echo "=== 部署完成 ==="
echo ""
echo "服务地址: http://localhost:11123"
echo "API 文档: http://localhost:11123/"
echo "服务版本: $VERSION ($GIT_COMMIT)"
echo ""
echo "查看日志: docker-compose logs -f"
echo "停止服务: docker-compose down"
echo "重启服务: docker-compose restart"
