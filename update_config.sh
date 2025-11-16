#!/bin/bash

# 更新配置脚本 - 根据下载的模型更新 config.yaml

set -e

MODELS_DIR="./models"
CONFIG_FILE="./config.yaml"
CONFIG_BACKUP="./config.yaml.bak"

echo "=== 更新配置文件 ==="
echo ""

# 备份原配置
if [ -f "$CONFIG_FILE" ]; then
    cp "$CONFIG_FILE" "$CONFIG_BACKUP"
    echo "✓ 已备份原配置到 $CONFIG_BACKUP"
fi

# 检测实时识别模型
STREAMING_MODEL=""
if [ -d "$MODELS_DIR/streaming/sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20" ]; then
    STREAMING_MODEL="sherpa-onnx-streaming-zipformer-bilingual-zh-en-2023-02-20"
elif [ -d "$MODELS_DIR/streaming/icefall-asr-zipformer-streaming-wenetspeech-20230615" ]; then
    STREAMING_MODEL="icefall-asr-zipformer-streaming-wenetspeech-20230615"
fi

# 检测离线识别模型
OFFLINE_MODEL=""
if [ -d "$MODELS_DIR/offline/sherpa-onnx-paraformer-zh-2023-09-14" ]; then
    OFFLINE_MODEL="sherpa-onnx-paraformer-zh-2023-09-14"
elif [ -d "$MODELS_DIR/offline/sherpa-onnx-whisper-tiny.en" ]; then
    OFFLINE_MODEL="sherpa-onnx-whisper-tiny.en"
fi

# 检测说话者分离模型
SEGMENTATION_MODEL=""
EMBEDDING_MODEL=""
if [ -d "$MODELS_DIR/diarization/sherpa-onnx-pyannote-segmentation-3-0" ]; then
    SEGMENTATION_MODEL="sherpa-onnx-pyannote-segmentation-3-0"
fi
if [ -f "$MODELS_DIR/diarization/3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx" ]; then
    EMBEDDING_MODEL="3dspeaker_speech_eres2net_base_sv_zh-cn_3dspeaker_16k.onnx"
fi

echo ""
echo "检测到的模型："
echo "  实时识别: ${STREAMING_MODEL:-未找到}"
echo "  离线识别: ${OFFLINE_MODEL:-未找到}"
echo "  说话者分割: ${SEGMENTATION_MODEL:-未找到}"
echo "  说话者嵌入: ${EMBEDDING_MODEL:-未找到}"
echo ""

# 更新配置文件
cat > "$CONFIG_FILE" << EOF
server:
  host: "0.0.0.0"
  port: 11123
  max_connections: 1000
  read_timeout: 60
  write_timeout: 60

# 实时语音识别配置（Streaming ASR）
streaming_asr:
  enabled: $([ -n "$STREAMING_MODEL" ] && echo "true" || echo "false")
  model_type: "zipformer"
  models_dir: "/models/streaming/${STREAMING_MODEL}"
  encoder: "exp/encoder-epoch-12-avg-4-chunk-16-left-128.onnx"
  decoder: "exp/decoder-epoch-12-avg-4-chunk-16-left-128.onnx"
  joiner: "exp/joiner-epoch-12-avg-4-chunk-16-left-128.onnx"
  tokens: "data/lang_char/tokens.txt"
  num_threads: 4
  sample_rate: 16000
  feature_dim: 80
  enable_endpoint: true
  rule1_min_trailing_silence: 2.4
  rule2_min_trailing_silence: 1.2
  rule3_min_utterance_length: 20

# 离线语音识别配置（Non-streaming ASR）
offline_asr:
  enabled: $([ -n "$OFFLINE_MODEL" ] && echo "true" || echo "false")
  model_type: "paraformer"
  models_dir: "/models/offline/${OFFLINE_MODEL}"
  encoder: "encoder.int8.onnx"
  decoder: "decoder.int8.onnx"
  tokens: "tokens.txt"
  num_threads: 4
  sample_rate: 16000
  decoding_method: "greedy_search"
  max_active_paths: 4

# 说话者分离配置（Speaker Diarization）
speaker_diarization:
  enabled: $([ -n "$SEGMENTATION_MODEL" ] && [ -n "$EMBEDDING_MODEL" ] && echo "true" || echo "false")
  models_dir: "/models/diarization"
  segmentation_model: "${SEGMENTATION_MODEL}/model.onnx"
  embedding_model: "${EMBEDDING_MODEL}"
  clustering:
    num_clusters: 0
    threshold: 0.5
  num_threads: 2

# VAD（语音活动检测）配置
vad:
  enabled: true
  model: "/models/vad/silero_vad.onnx"
  sample_rate: 16000
  min_silence_duration: 500
  min_speech_duration: 250
  threshold: 0.5
  window_size: 512
  num_threads: 1

# 并发控制
concurrency:
  max_streaming_sessions: 100
  max_offline_jobs: 50
  worker_pool_size: 20
  queue_size: 1000

# 日志配置
logging:
  level: "info"
  file: "/logs/airecorder.log"
  max_size: 100
  max_backups: 5
  max_age: 30
EOF

echo "✓ 配置文件已更新"
echo ""
echo "已启用的功能："
grep "enabled: true" "$CONFIG_FILE" | while read line; do
    echo "  $line"
done

echo ""
echo "配置文件位置: $CONFIG_FILE"
echo "备份文件位置: $CONFIG_BACKUP"
echo ""
echo "如需自定义配置，请编辑 $CONFIG_FILE"
