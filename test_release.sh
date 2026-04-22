#!/bin/bash

# AI Recorder 发布测试脚本
# 用于测试 make release 的完整流程

set -e

echo "=========================================="
echo "  AI Recorder 发布流程测试"
echo "=========================================="
echo ""

echo "📋 检查前置条件..."
echo ""

# 检查模型
if [ ! -f "./models/vad/silero_vad.onnx" ]; then
    echo "❌ 模型文件不存在"
    echo "请先运行: make download-models"
    exit 1
fi
echo "✓ 模型文件存在"

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装"
    exit 1
fi
echo "✓ Docker 已安装"

# 检查 scripts 目录
if [ ! -d "scripts" ]; then
    echo "❌ scripts 目录不存在"
    exit 1
fi
echo "✓ scripts 目录存在"

# 检查必需的脚本
for script in deploy.sh verify.sh test_env.sh README.md; do
    if [ ! -f "scripts/$script" ]; then
        echo "❌ scripts/$script 不存在"
        exit 1
    fi
done
echo "✓ 所有脚本文件存在"

echo ""
echo "🎯 开始测试发布流程..."
echo ""

# 模拟 make release 的关键步骤
echo "步骤 1: 复制脚本文件..."
mkdir -p offline_deploy
cp scripts/*.sh offline_deploy/
cp scripts/README.md offline_deploy/
chmod +x offline_deploy/*.sh
echo "✓ 脚本文件已复制"

echo ""
echo "步骤 2: 检查 offline_deploy 内容..."
ls -lh offline_deploy/
echo ""

echo "步骤 3: 验证文件..."
cd offline_deploy
for file in deploy.sh verify.sh test_env.sh README.md; do
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "❌ $file 缺失"
        exit 1
    fi
done
cd ..

echo ""
echo "=========================================="
echo "✓ 测试通过！"
echo "=========================================="
echo ""
echo "📦 offline_deploy 目录已准备就绪"
echo "   包含所有必需的部署脚本"
echo ""
echo "🚀 现在可以运行: make release"
echo ""
