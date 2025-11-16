#!/bin/bash

# 编译 ARM64 二进制文件脚本
# 用于生成可在 Linux ARM64 环境运行的程序

set -e

# 读取版本信息
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "=== 编译 AI Recorder 二进制文件 ==="
echo "版本: $VERSION"
echo "Git提交: $GIT_COMMIT"
echo "构建时间: $BUILD_TIME"
echo "目标平台: Linux ARM64"
echo ""

# 创建输出目录
mkdir -p bin/lib

# 使用 Docker 进行交叉编译（确保环境一致性）
echo "使用 Docker 进行交叉编译..."
docker run --rm \
  --platform linux/arm64 \
  -v "$PWD":/build \
  -w /build \
  golang:1.21-bookworm \
  bash -c "
    echo '安装构建依赖...'
    apt-get update && apt-get install -y gcc g++ > /dev/null 2>&1
    
    echo '下载 Go 依赖...'
    go mod download
    
    echo '编译程序...'
    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build \
      -o bin/airecorder \
      -ldflags=\"-s -w \
      -X 'airecorder/internal/version.Version=${VERSION}' \
      -X 'airecorder/internal/version.GitCommit=${GIT_COMMIT}' \
      -X 'airecorder/internal/version.BuildTime=${BUILD_TIME}'\" \
      main.go
    
    echo '复制共享库...'
    cp /go/pkg/mod/github.com/k2-fsa/sherpa-onnx-go-linux@v1.12.17/lib/aarch64-unknown-linux-gnu/*.so* /build/bin/lib/ 2>/dev/null || true
    
    echo '设置文件权限...'
    chmod +x /build/bin/airecorder
    chown -R $(id -u):$(id -g) /build/bin
  "

# 检查编译结果
if [ -f "./bin/airecorder" ]; then
    echo ""
    echo "✓ 编译成功！"
    echo "二进制文件: ./bin/airecorder"
    echo "共享库目录: ./bin/lib/"
    echo ""
    
    # 显示文件大小
    echo "文件大小:"
    ls -lh ./bin/airecorder
    echo ""
    
    # 计算 MD5
    if command -v md5sum &> /dev/null; then
        echo "MD5 校验值:"
        md5sum ./bin/airecorder | tee ./bin/airecorder.md5
    elif command -v md5 &> /dev/null; then
        echo "MD5 校验值:"
        md5 -r ./bin/airecorder | tee ./bin/airecorder.md5
    fi
    
    echo ""
    echo "现在可以将 ./bin 目录复制到服务器"
    echo "然后使用 docker-compose -f docker-compose.runtime.yml up -d 启动"
else
    echo "✗ 编译失败"
    exit 1
fi
