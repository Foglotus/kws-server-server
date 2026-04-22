#!/bin/bash

##############################################################################
# AI Recorder 验证脚本
# 用途：验证部署包的完整性
##############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 读取版本信息
VERSION=$(cat "$SCRIPT_DIR/VERSION" 2>/dev/null || echo "unknown")

echo "=========================================="
echo "  AI Recorder 部署包验证"
echo "  版本: $VERSION"
echo "=========================================="
echo ""

# 检查必需文件
echo "检查必需文件..."
REQUIRED_FILES=(
    "VERSION"
    "prepare.sh"
    "deploy.sh"
    "test_env.sh"
    "verify.sh"
)

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$SCRIPT_DIR/$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file (缺失)"
        MISSING_FILES+=("$file")
    fi
done

# 检查生成的文件
echo ""
echo "检查生成的部署文件..."
GENERATED_FILES=(
    "airecorder.tar.gz"
    "models.tar.gz"
    "config.yaml"
)

for file in "${GENERATED_FILES[@]}"; do
    if [ -f "$SCRIPT_DIR/$file" ]; then
        SIZE=$(du -h "$SCRIPT_DIR/$file" | cut -f1)
        echo "✓ $file ($SIZE)"
    else
        echo "⚠ $file (未生成，请先运行 prepare.sh)"
    fi
done

# 检查模型文件（如果已解压）
echo ""
echo "检查 AI 模型文件..."
if [ -d "$SCRIPT_DIR/../models" ]; then
    MODELS_DIR="$SCRIPT_DIR/../models"
    REQUIRED_MODELS=(
        "vad/silero_vad.onnx"
        "streaming/sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20"
        "offline/sherpa-onnx-paraformer-zh-2023-09-14"
        "diarization/sherpa-onnx-pyannote-segmentation-3-0"
        "diarization/3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx"
        "punctuation/sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12"
    )
    
    MISSING_MODELS=()
    for model in "${REQUIRED_MODELS[@]}"; do
        if [ -e "$MODELS_DIR/$model" ]; then
            echo "✓ $model"
        else
            echo "✗ $model (缺失)"
            MISSING_MODELS+=("$model")
        fi
    done
    
    if [ ${#MISSING_MODELS[@]} -gt 0 ]; then
        echo "⚠ 缺少 ${#MISSING_MODELS[@]} 个模型文件"
        MISSING_FILES+=("${MISSING_MODELS[@]}")
    fi
elif [ -f "$SCRIPT_DIR/models.tar.gz" ]; then
    echo "✓ models.tar.gz 存在 (模型未解压)"
else
    echo "⚠ 模型文件或模型压缩包不存在"
fi

# 验证校验和
echo ""
if [ -f "$SCRIPT_DIR/checksums.md5" ]; then
    echo "验证文件校验和..."
    cd "$SCRIPT_DIR"
    if command -v md5sum &> /dev/null; then
        md5sum -c checksums.md5
    elif command -v md5 &> /dev/null; then
        echo "⚠ macOS 系统，跳过自动校验"
    fi
else
    echo "⚠ 校验和文件不存在"
fi

echo ""
if [ ${#MISSING_FILES[@]} -eq 0 ]; then
    echo "=========================================="
    echo "✓ 验证通过"
    echo "=========================================="
    echo ""
    echo "📦 部署包信息:"
    echo "  - 版本号: $VERSION"
    if [ -f "$SCRIPT_DIR/MANIFEST.txt" ]; then
        echo "  - 详细信息请查看: MANIFEST.txt"
    fi
    echo ""
    echo "✅ 部署包已准备就绪，可以执行部署"
    echo "   运行: ./deploy.sh"
else
    echo "=========================================="
    echo "✗ 验证失败"
    echo "=========================================="
    echo ""
    echo "缺少 ${#MISSING_FILES[@]} 个文件:"
    for file in "${MISSING_FILES[@]}"; do
        echo "  - $file"
    done
    echo ""
    echo "❌ 请先运行 prepare.sh 准备部署包"
fi
echo ""
    echo "✗ 验证失败: 缺少 ${#MISSING_FILES[@]} 个必需文件"
    echo "=========================================="
    exit 1
fi
