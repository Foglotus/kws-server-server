#!/bin/bash

# AI Recorder 模型下载脚本
# 用于下载 sherpa-onnx 模型文件

set -e

MODELS_DIR="./models"

echo "=== AI Recorder 模型下载工具 ==="
echo ""

# 创建模型目录
mkdir -p $MODELS_DIR/streaming
mkdir -p $MODELS_DIR/offline
mkdir -p $MODELS_DIR/diarization
mkdir -p $MODELS_DIR/vad
mkdir -p $MODELS_DIR/punctuation

# 下载 VAD 模型
echo "下载 VAD 模型..."
if [ ! -f "$MODELS_DIR/vad/silero_vad.onnx" ]; then
    curl -SL -o $MODELS_DIR/vad/silero_vad.onnx \
        https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/silero_vad.onnx
    echo "✓ VAD 模型下载完成"
else
    echo "✓ VAD 模型已存在"
fi

# 下载流式识别模型（中英双语）
echo ""
echo "下载流式识别模型（中英双语 Zipformer）..."
STREAMING_MODEL="sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20"
if [ ! -d "$MODELS_DIR/streaming/$STREAMING_MODEL" ]; then
    cd $MODELS_DIR/streaming
    curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/${STREAMING_MODEL}.tar.bz2
    tar xvf ${STREAMING_MODEL}.tar.bz2
    rm ${STREAMING_MODEL}.tar.bz2
    cd ../..
    echo "✓ 流式识别模型下载完成"
else
    echo "✓ 流式识别模型已存在"
fi

# 下载离线识别模型（Paraformer 中文）
echo ""
echo "下载离线识别模型（Paraformer 中文）..."
OFFLINE_MODEL="sherpa-onnx-paraformer-zh-2023-09-14"
if [ ! -d "$MODELS_DIR/offline/$OFFLINE_MODEL" ]; then
    cd $MODELS_DIR/offline
    curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/${OFFLINE_MODEL}.tar.bz2
    tar xvf ${OFFLINE_MODEL}.tar.bz2
    rm ${OFFLINE_MODEL}.tar.bz2
    cd ../..
    echo "✓ 离线识别模型下载完成"
else
    echo "✓ 离线识别模型已存在"
fi

# 下载说话者分离模型
echo ""
echo "下载说话者分割模型..."
SEGMENTATION_MODEL="sherpa-onnx-pyannote-segmentation-3-0"
if [ ! -d "$MODELS_DIR/diarization/$SEGMENTATION_MODEL" ]; then
    cd $MODELS_DIR/diarization
    curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-segmentation-models/${SEGMENTATION_MODEL}.tar.bz2
    tar xvf ${SEGMENTATION_MODEL}.tar.bz2
    rm ${SEGMENTATION_MODEL}.tar.bz2
    cd ../..
    echo "✓ 说话者分割模型下载完成"
else
    echo "✓ 说话者分割模型已存在"
fi

echo ""
echo "下载说话者嵌入模型..."
EMBEDDING_MODEL="3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx"
if [ ! -f "$MODELS_DIR/diarization/$EMBEDDING_MODEL" ]; then
    cd $MODELS_DIR/diarization
    curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-recongition-models/${EMBEDDING_MODEL}
    cd ../..
    echo "✓ 说话者嵌入模型下载完成"
else
    echo "✓ 说话者嵌入模型已存在"
fi

# 下载标点符号模型
echo ""
echo "下载标点符号模型（中英双语）..."
PUNCT_MODEL="sherpa-onnx-punct-ct-transformer-zh-en-vocab272727-2024-04-12"
if [ ! -d "$MODELS_DIR/punctuation/$PUNCT_MODEL" ]; then
    cd $MODELS_DIR/punctuation
    curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/punctuation-models/${PUNCT_MODEL}.tar.bz2
    tar xvf ${PUNCT_MODEL}.tar.bz2
    rm ${PUNCT_MODEL}.tar.bz2
    cd ../..
    echo "✓ 标点符号模型下载完成"
else
    echo "✓ 标点符号模型已存在"
fi

echo ""
echo "=== 所有模型下载完成 ==="
echo ""
echo "模型文件位置："
echo "  - VAD: $MODELS_DIR/vad/"
echo "  - 流式识别: $MODELS_DIR/streaming/"
echo "  - 离线识别: $MODELS_DIR/offline/"
echo "  - 说话者分离: $MODELS_DIR/diarization/"
echo "  - 标点符号: $MODELS_DIR/punctuation/"
echo ""
echo "现在可以运行: docker-compose up -d"
