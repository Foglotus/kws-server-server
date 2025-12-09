# AI Recorder - 语音识别服务

基于 ARM 平台 sherpa-onnx 库架构实现的 Docker 发布语音识别服务。

## 功能特性

### ✨ 核心功能

1. **实时语音识别 (Streaming ASR)**
   - WebSocket 接口，支持实时音频流识别
   - 端点检测，自动分段
   - 低延迟，高并发支持
   
2. **离线语音识别 (Offline ASR)**
   - HTTP POST 接口，支持文件上传或 Base64 编码音频
   - **多格式支持**: 自动检测和转换 WAV、MP3、M4A、FLAC、OGG 等格式
   - 支持多种模型：Paraformer、Whisper 等
   - 两种模式：
     - **非说话者模式**: 直接输出完整文本
     - **说话者模式**: 自动区分说话者并标注

3. **说话者分离 (Speaker Diarization)**
   - 自动检测和分离多个说话者
   - 为每个说话者片段提供时间戳
   - 可独立使用或与 ASR 结合

### 🚀 技术特性

- **高并发**: 支持多人多场景同时使用
- **ARM 优化**: 专为 ARM64 架构优化
- **Docker 部署**: 一键部署，易于管理
- **模块化设计**: 可灵活启用/禁用各功能模块
- **健康监控**: 内置健康检查和统计接口
- **多格式支持**: 自动识别并转换常见音频格式（需 FFmpeg）

## 系统架构

```
┌─────────────────────────────────────────────────┐
│                   Client                         │
│          (Web/Mobile/API Consumer)              │
└────────────────┬────────────────────────────────┘
                 │
                 │ HTTP/WebSocket
                 ▼
┌─────────────────────────────────────────────────┐
│              API Gateway (Gin)                   │
│    ┌──────────┬──────────┬──────────────────┐  │
│    │ /streaming│ /offline │  /diarization   │  │
│    │   /asr    │   /asr   │                 │  │
│    └──────────┴──────────┴──────────────────┘  │
└─────────┬────────────┬────────────┬─────────────┘
          │            │            │
          ▼            ▼            ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  Streaming   │ │   Offline    │ │ Diarization  │
│ ASR Manager  │ │ ASR Manager  │ │   Manager    │
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │
       └────────────────┴────────────────┘
                        │
                        ▼
              ┌──────────────────┐
              │  sherpa-onnx     │
              │   (ONNX Runtime) │
              └──────────────────┘
                        │
                        ▼
              ┌──────────────────┐
              │    AI Models     │
              │  (ONNX format)   │
              └──────────────────┘
```

## 快速开始

### 环境要求

- Docker & Docker Compose
- ARM64 架构 (Apple Silicon, Raspberry Pi 4+, 等)
- 至少 4GB 内存
- 至少 10GB 磁盘空间（用于模型文件）

### 安装步骤

#### 1. 克隆项目

```bash
git clone <repository-url>
cd airecorder
```

#### 2. 下载模型文件

```bash
chmod +x download_models.sh
./download_models.sh
```

模型会下载到 `./models/` 目录：
- `models/vad/` - VAD 模型
- `models/streaming/` - 实时识别模型
- `models/offline/` - 离线识别模型
- `models/diarization/` - 说话者分离模型

#### 3. 配置服务

编辑 `config.yaml` 根据需要调整配置：

```yaml
server:
  host: "0.0.0.0"
  port: 11123
  
streaming_asr:
  enabled: true
  num_threads: 4
  
offline_asr:
  enabled: true
  num_threads: 4
  
speaker_diarization:
  enabled: true
```

#### 4. 部署服务

```bash
chmod +x deploy.sh
./deploy.sh
```

或手动部署：

```bash
# 构建镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

#### 5. 验证服务

```bash
# 健康检查
curl http://localhost:11123/health

# 查看 API 信息
curl http://localhost:11123/
```

#### 6. 使用 Web 测试页面 ⭐推荐

在浏览器中打开测试控制台：

```
http://localhost:11123/test
```

**功能特性：**

- 🎤 **实时语音识别测试** - 直接使用浏览器麦克风进行实时录音和识别
- 📁 **离线语音识别测试** - 上传音频文件进行批量识别
- 👥 **说话者分离测试** - 多人对话场景的说话者识别
- 🔧 **API 端点测试** - 一键测试所有 API 接口
- 📊 **服务状态监控** - 实时查看服务健康状态

这是最简单的测试方式，无需编写代码或安装任何工具，直接在浏览器中即可完成所有功能测试！

## API 使用指南

### 1. 实时语音识别 (WebSocket)

**端点**: `ws://localhost:11123/api/v1/streaming/asr`

**消息格式**:

发送音频数据：
```json
{
  "type": "audio",
  "audio": "<Base64编码的PCM音频>",
  "sample_rate": 16000
}
```

控制命令：
```json
{
  "type": "control",
  "command": "reset"  // 或 "stop"
}
```

**响应格式**:

```json
{
  "type": "partial",  // 或 "result", "error"
  "text": "识别的文本",
  "is_endpoint": false,
  "segment": 0
}
```

**示例代码** (JavaScript):

```javascript
const ws = new WebSocket('ws://localhost:11123/api/v1/streaming/asr');

ws.onopen = () => {
  console.log('连接成功');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('识别结果:', data.text);
};

// 发送音频数据
function sendAudio(audioBuffer) {
  const base64Audio = btoa(String.fromCharCode(...new Uint8Array(audioBuffer)));
  ws.send(JSON.stringify({
    type: 'audio',
    audio: base64Audio,
    sample_rate: 16000
  }));
}
```

### 2. 离线语音识别

**端点**: `POST /api/v1/offline/asr`

**请求格式**:

方式 1: JSON + Base64
```json
{
  "audio": "<Base64编码的音频数据>",
  "sample_rate": 16000
}
```

方式 2: 文件上传
```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@test.wav"
```

**响应格式**:

```json
{
  "text": "识别的完整文本",
  "duration": 5.2
}
```

### 3. 带说话者分离的识别

**端点**: `POST /api/v1/offline/asr/diarization`

**请求格式**: 同离线识别

**响应格式**:

```json
{
  "text": "完整文本内容",
  "segments": [
    {
      "start": 0.0,
      "end": 2.5,
      "speaker": 0,
      "text": "第一个说话者的内容"
    },
    {
      "start": 2.5,
      "end": 5.0,
      "speaker": 1,
      "text": "第二个说话者的内容"
    }
  ],
  "duration": 5.0
}
```

### 4. 独立说话者分离

**端点**: `POST /api/v1/diarization`

**响应格式**:

```json
{
  "segments": [
    {
      "start": 0.0,
      "end": 2.5,
      "speaker": 0
    }
  ],
  "duration": 5.0
}
```

### 5. 统计信息

**端点**: `GET /api/v1/stats`

**响应格式**:

```json
{
  "streaming": {
    "active_sessions": 5,
    "total_sessions": 127,
    "total_audio_frames": 50000
  },
  "offline": {
    "total_requests": 89,
    "success_count": 87,
    "failure_count": 2
  }
}
```

## 测试示例

### Python 测试脚本

```python
import requests
import base64
import json

# 读取音频文件
with open('test.wav', 'rb') as f:
    audio_data = f.read()

# Base64 编码
audio_base64 = base64.b64encode(audio_data).decode('utf-8')

# 离线识别
response = requests.post(
    'http://localhost:11123/api/v1/offline/asr',
    json={
        'audio': audio_base64,
        'sample_rate': 16000
    }
)

print('识别结果:', response.json())

# 带说话者分离
response = requests.post(
    'http://localhost:11123/api/v1/offline/asr/diarization',
    json={
        'audio': audio_base64,
        'sample_rate': 16000
    }
)

print('说话者分离结果:', json.dumps(response.json(), ensure_ascii=False, indent=2))
```

### WebSocket 测试 (Node.js)

```javascript
const WebSocket = require('ws');
const fs = require('fs');

const ws = new WebSocket('ws://localhost:11123/api/v1/streaming/asr');

ws.on('open', function open() {
  console.log('连接已建立');
  
  // 读取音频文件并分块发送
  const audioBuffer = fs.readFileSync('test.wav');
  const chunkSize = 3200; // 0.1秒的音频数据 (16000Hz * 2 bytes)
  
  for (let i = 0; i < audioBuffer.length; i += chunkSize) {
    const chunk = audioBuffer.slice(i, i + chunkSize);
    const base64Chunk = chunk.toString('base64');
    
    ws.send(JSON.stringify({
      type: 'audio',
      audio: base64Chunk,
      sample_rate: 16000
    }));
    
    // 模拟实时流
    setTimeout(() => {}, 100);
  }
});

ws.on('message', function message(data) {
  const result = JSON.parse(data);
  console.log('识别结果:', result.text);
});

ws.on('close', function close() {
  console.log('连接已关闭');
});
```

## 配置说明

### 服务器配置

```yaml
server:
  host: "0.0.0.0"          # 监听地址
  port: 11123                # 监听端口
  max_connections: 1000     # 最大连接数
  read_timeout: 60          # 读超时（秒）
  write_timeout: 60         # 写超时（秒）
```

### 实时识别配置

```yaml
streaming_asr:
  enabled: true             # 是否启用
  model_type: "zipformer"   # 模型类型
  models_dir: "/models/streaming"
  num_threads: 4            # 推理线程数
  sample_rate: 16000        # 采样率
  enable_endpoint: true     # 启用端点检测
```

### 离线识别配置

```yaml
offline_asr:
  enabled: true
  model_type: "paraformer"  # whisper, paraformer, transducer
  models_dir: "/models/offline"
  num_threads: 4
  sample_rate: 16000
  decoding_method: "greedy_search"
```

### 说话者分离配置

```yaml
speaker_diarization:
  enabled: true
  models_dir: "/models/diarization"
  clustering:
    num_clusters: 0         # 0=自动检测
    threshold: 0.5          # 聚类阈值
```

### 并发控制

```yaml
concurrency:
  max_streaming_sessions: 100   # 最大实时会话数
  max_offline_jobs: 50          # 最大离线任务数
  worker_pool_size: 20          # 工作线程池大小
  queue_size: 1000              # 队列大小
```

## 性能优化

### 1. 线程配置

根据 CPU 核心数调整：
- 单核心: `num_threads: 1`
- 双核心: `num_threads: 2`
- 四核心+: `num_threads: 4`

### 2. 并发限制

根据内存和 CPU 调整：
```yaml
concurrency:
  max_streaming_sessions: 50   # 减少同时会话数
  max_offline_jobs: 20          # 减少离线任务数
```

### 3. Docker 资源限制

在 `docker-compose.yml` 中调整：
```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 8G
```

## 故障排除

### 1. 服务无法启动

检查日志：
```bash
docker-compose logs -f airecorder
```

常见问题：
- 模型文件未下载或路径错误
- 端口 11123 被占用
- 内存不足

### 2. 识别结果不准确

- 确保音频采样率为 16000Hz
- 检查音频格式（推荐 PCM 16-bit）
- 尝试不同的模型

### 3. 性能问题

- 增加 `num_threads`
- 减少并发限制
- 增加 Docker 内存限制

### 4. WebSocket 连接失败

- 检查防火墙设置
- 确认 WebSocket 升级支持
- 查看浏览器控制台错误

## 维护和监控

### 查看日志

```bash
# 实时日志
docker-compose logs -f

# 最近 100 行
docker-compose logs --tail=100

# 特定服务
docker-compose logs airecorder
```

### 重启服务

```bash
# 重启
docker-compose restart

# 停止
docker-compose down

# 启动
docker-compose up -d
```

### 更新服务

```bash
# 拉取最新代码
git pull

# 重新构建
docker-compose build

# 重新部署
docker-compose up -d
```

### 监控资源使用

```bash
# Docker 统计
docker stats airecorder

# 磁盘使用
docker system df
```

## 高级功能

### 1. 自定义模型

将自己的模型文件放到对应目录，并更新 `config.yaml`：

```yaml
streaming_asr:
  models_dir: "/models/streaming/my-custom-model"
  encoder: "custom-encoder.onnx"
  decoder: "custom-decoder.onnx"
  joiner: "custom-joiner.onnx"
  tokens: "custom-tokens.txt"
```

### 2. 负载均衡

使用 Nginx 或其他负载均衡器：

```nginx
upstream airecorder {
    server 192.168.1.10:11123;
    server 192.168.1.11:11123;
    server 192.168.1.12:11123;
}

server {
    listen 80;
    
    location / {
        proxy_pass http://airecorder;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 3. 集群部署

使用 Docker Swarm 或 Kubernetes 进行集群部署。

## 开发指南

### 项目结构

```
airecorder/
├── main.go                 # 主入口
├── go.mod                  # Go 依赖
├── config.yaml             # 配置文件
├── Dockerfile              # Docker 构建文件
├── docker-compose.yml      # Docker Compose 配置
├── internal/
│   ├── config/            # 配置管理
│   ├── server/            # 服务器
│   ├── handler/           # HTTP/WebSocket 处理器
│   └── asr/               # ASR 核心逻辑
│       ├── streaming.go   # 实时识别
│       ├── offline.go     # 离线识别
│       └── diarization.go # 说话者分离
├── models/                # 模型文件目录
└── logs/                  # 日志目录
```

### 本地开发

```bash
# 安装依赖
go mod download

# 运行服务
go run main.go

# 构建
go build -o airecorder main.go
```

## 许可证

本项目基于 MIT 许可证开源。

## 致谢

- [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx) - 核心语音识别库
- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket 支持

## 联系方式

如有问题或建议，请提交 Issue 或 Pull Request。
