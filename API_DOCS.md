# API 接口文档

## 基础信息

- **基础 URL**: `http://localhost:11123`
- **API 版本**: v1
- **数据格式**: JSON
- **字符编码**: UTF-8
- **测试页面**: `http://localhost:11123/test`

## 认证

当前版本不需要认证。生产环境建议添加认证机制。

---

## 0. Web 测试页面

### GET /test

访问 Web 测试控制台，提供可视化的服务测试界面

**访问方式**:

在浏览器中打开: `http://localhost:11123/test`

**功能特性**:
- 实时语音识别测试（WebSocket）
- 离线语音识别测试（文件上传）
- 说话者分离测试
- API 端点测试
- 服务健康状态监控

**说明**: 这是一个完整的 Web 应用，无需安装任何客户端，直接在浏览器中即可测试所有 API 功能。

---

## 1. 健康检查

### GET /health

检查服务健康状态

**请求示例**:

```bash
curl http://localhost:11123/health
```

**响应示例**:

```json
{
  "status": "healthy",
  "service": "airecorder"
}
```

**状态码**:
- `200`: 服务正常
- `503`: 服务不可用

---

## 2. 服务信息

### GET /

获取服务基本信息和可用端点

**请求示例**:

```bash
curl http://localhost:11123/
```

**响应示例**:

```json
{
  "service": "AI Recorder - Speech Recognition Service",
  "version": "1.0.0",
  "endpoints": {
    "streaming_asr": "/api/v1/streaming/asr (WebSocket)",
    "offline_asr": "/api/v1/offline/asr (POST)",
    "offline_with_diarization": "/api/v1/offline/asr/diarization (POST)",
    "diarization": "/api/v1/diarization (POST)",
    "stats": "/api/v1/stats (GET)"
  }
}
```

---

## 3. 实时语音识别 (WebSocket)

### WS /api/v1/streaming/asr

实时语音识别，支持持续音频流

**连接示例**:

```javascript
const ws = new WebSocket('ws://localhost:11123/api/v1/streaming/asr');
```

### 消息格式

#### 发送音频数据

```json
{
  "type": "audio",
  "audio": "<Base64编码的PCM音频数据>",
  "sample_rate": 16000
}
```

**字段说明**:
- `type`: 消息类型，固定为 "audio"
- `audio`: Base64 编码的 PCM 16-bit 音频数据
- `sample_rate`: 采样率（默认 16000Hz）

#### 控制命令

```json
{
  "type": "control",
  "command": "reset"
}
```

**支持的命令**:
- `reset`: 重置识别状态
- `stop`: 停止识别并关闭连接

### 响应格式

#### 部分结果

```json
{
  "type": "partial",
  "text": "识别中的文本",
  "is_endpoint": false,
  "segment": 0
}
```

#### 完整结果

```json
{
  "type": "result",
  "text": "完整的识别文本",
  "is_endpoint": true,
  "segment": 1
}
```

#### 错误消息

```json
{
  "type": "error",
  "error": "错误描述"
}
```

**字段说明**:
- `type`: 响应类型 (partial/result/error)
- `text`: 识别的文本
- `is_endpoint`: 是否检测到语音端点
- `segment`: 当前片段序号
- `error`: 错误信息（仅错误时）

---

## 4. 离线语音识别

### POST /api/v1/offline/asr

离线批量语音识别，不区分说话者

#### 方式 1: JSON + Base64

**请求示例**:

```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -H "Content-Type: application/json" \
  -d '{
    "audio": "<Base64编码的音频数据>",
    "sample_rate": 16000
  }'
```

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| audio | string | 是 | Base64 编码的音频数据 |
| sample_rate | int | 否 | 采样率，默认 16000 |

#### 方式 2: 文件上传

**请求示例**:

```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@test.wav"
```

**响应示例**:

```json
{
  "text": "这是识别出的完整文本内容",
  "duration": 5.2
}
```

**响应字段**:

| 字段 | 类型 | 说明 |
|------|------|------|
| text | string | 识别结果文本 |
| duration | float | 音频时长（秒） |
| error | string | 错误信息（仅失败时） |

**状态码**:
- `200`: 成功
- `400`: 请求参数错误
- `500`: 服务器内部错误

---

## 5. 带说话者分离的识别

### POST /api/v1/offline/asr/diarization

离线语音识别 + 说话者分离，自动区分不同说话者

**请求格式**: 同离线识别

**请求示例**:

```bash
curl -X POST http://localhost:11123/api/v1/offline/asr/diarization \
  -H "Content-Type: application/json" \
  -d '{
    "audio": "<Base64编码的音频数据>",
    "sample_rate": 16000
  }'
```

**响应示例**:

```json
{
  "text": "完整文本内容 包含所有说话者的内容",
  "segments": [
    {
      "start": 0.0,
      "end": 2.5,
      "speaker": 0,
      "text": "第一个说话者说的内容"
    },
    {
      "start": 2.5,
      "end": 5.0,
      "speaker": 1,
      "text": "第二个说话者说的内容"
    },
    {
      "start": 5.0,
      "end": 7.8,
      "speaker": 0,
      "text": "第一个说话者继续说"
    }
  ],
  "duration": 7.8
}
```

**响应字段**:

| 字段 | 类型 | 说明 |
|------|------|------|
| text | string | 完整文本（所有说话者） |
| segments | array | 说话者片段数组 |
| segments[].start | float | 片段开始时间（秒） |
| segments[].end | float | 片段结束时间（秒） |
| segments[].speaker | int | 说话者 ID (0, 1, 2, ...) |
| segments[].text | string | 该片段的识别文本 |
| duration | float | 总时长（秒） |
| error | string | 错误信息（仅失败时） |

**状态码**:
- `200`: 成功
- `400`: 请求参数错误
- `500`: 服务器内部错误

---

## 6. 独立说话者分离

### POST /api/v1/diarization

仅进行说话者分离，不进行语音识别

**请求示例**:

```bash
curl -X POST http://localhost:11123/api/v1/diarization \
  -H "Content-Type: application/json" \
  -d '{
    "audio": "<Base64编码的音频数据>",
    "sample_rate": 16000
  }'
```

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| audio | string | 是 | Base64 编码的音频数据 |
| sample_rate | int | 否 | 采样率，默认 16000 |

**响应示例**:

```json
{
  "segments": [
    {
      "start": 0.0,
      "end": 2.5,
      "speaker": 0
    },
    {
      "start": 2.5,
      "end": 5.0,
      "speaker": 1
    },
    {
      "start": 5.0,
      "end": 7.8,
      "speaker": 0
    }
  ],
  "duration": 7.8
}
```

**响应字段**:

| 字段 | 类型 | 说明 |
|------|------|------|
| segments | array | 说话者片段数组 |
| segments[].start | float | 片段开始时间（秒） |
| segments[].end | float | 片段结束时间（秒） |
| segments[].speaker | int | 说话者 ID (0, 1, 2, ...) |
| duration | float | 总时长（秒） |

**状态码**:
- `200`: 成功
- `400`: 请求参数错误
- `500`: 服务器内部错误

---

## 7. 统计信息

### GET /api/v1/stats

获取服务运行统计信息

**请求示例**:

```bash
curl http://localhost:11123/api/v1/stats
```

**响应示例**:

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

**响应字段**:

| 字段 | 类型 | 说明 |
|------|------|------|
| streaming.active_sessions | int | 当前活跃的实时会话数 |
| streaming.total_sessions | int | 累计实时会话总数 |
| streaming.total_audio_frames | int | 累计处理的音频帧数 |
| offline.total_requests | int | 离线识别请求总数 |
| offline.success_count | int | 成功的请求数 |
| offline.failure_count | int | 失败的请求数 |

**状态码**:
- `200`: 成功

---

## 错误码

| HTTP 状态码 | 说明 |
|------------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 端点不存在 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

## 错误响应格式

```json
{
  "error": "错误描述信息"
}
```

---

## 音频格式要求

### 支持的格式

- **编码**: PCM (未压缩)
- **采样率**: 16000 Hz (推荐)
- **位深**: 16-bit
- **声道**: 单声道 (Mono)
- **字节序**: 小端序 (Little-endian)

### 转换示例

使用 FFmpeg 转换音频格式：

```bash
ffmpeg -i input.mp3 -ar 16000 -ac 1 -f s16le -acodec pcm_s16le output.wav
```

---

## 速率限制

当前版本没有速率限制，但有并发限制：

- 最大实时会话数: 100 (可配置)
- 最大离线任务数: 50 (可配置)

达到限制时会返回错误。

---

## 最佳实践

### 1. 实时识别

- 每次发送 0.1-0.2 秒的音频数据
- 及时处理响应结果
- 使用 `reset` 命令清理状态

### 2. 离线识别

- 对于长音频，建议先进行分段
- 使用文件上传方式更高效
- 检查音频格式是否符合要求

### 3. 说话者分离

- 音频时长建议在 10 秒到 10 分钟之间
- 说话者数量建议不超过 10 人
- 确保音频质量良好

---

## 示例代码

完整示例代码请参考：
- Python: `test_api.py`
- WebSocket: `test_websocket.py`
- JavaScript: README 中的示例

---

**版本**: 1.0.0  
**更新时间**: 2025-01-13
