# 音频格式支持说明

## 概述

airecorder 现在支持多种常见音频格式的自动检测和转换，无需手动预处理音频文件。

## 支持的格式

### 直接支持（无需外部依赖）

- **WAV** - Wave Audio File Format
  - 支持 8-bit、16-bit、32-bit PCM
  - 支持单声道和多声道（自动转为单声道）
  - 支持任意采样率（自动重采样至 16kHz）

### 通过 FFmpeg 支持

当系统安装了 FFmpeg 时，额外支持：

| 格式 | 描述 | 常见扩展名 |
|------|------|-----------|
| MP3 | MPEG Audio Layer 3 | .mp3 |
| M4A | MPEG-4 Audio | .m4a, .mp4 |
| FLAC | Free Lossless Audio Codec | .flac |
| OGG | Ogg Vorbis | .ogg |
| OPUS | Opus Audio Codec | .opus |
| AAC | Advanced Audio Coding | .aac |
| WMA | Windows Media Audio | .wma |
| AMR | Adaptive Multi-Rate | .amr |

## 安装 FFmpeg

### macOS
```bash
brew install ffmpeg
```

### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install ffmpeg
```

### CentOS/RHEL
```bash
sudo yum install epel-release
sudo yum install ffmpeg
```

### Docker
如果使用 Docker，在 Dockerfile 中添加：
```dockerfile
RUN apt-get update && apt-get install -y ffmpeg
```

## 工作原理

1. **格式检测**：自动检测上传的音频文件格式（通过文件头魔数）
2. **格式转换**：
   - WAV 格式：直接解析，无需外部工具
   - 其他格式：使用 FFmpeg 转换为 PCM
3. **音频处理**：
   - 重采样至 16kHz（如果需要）
   - 转换为单声道（如果需要）
   - 转换为 float32 样本数组

## 使用示例

### 1. 上传 WAV 文件
```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@recording.wav"
```

### 2. 上传 MP3 文件
```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@podcast.mp3"
```

### 3. 上传 M4A 文件（iPhone 录音）
```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@voice_memo.m4a"
```

### 4. 上传 FLAC 文件（高质量无损）
```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@music.flac"
```

### 5. Python 示例
```python
import requests

# 上传任意格式的音频文件
with open('recording.mp3', 'rb') as f:
    files = {'audio_file': f}
    response = requests.post(
        'http://localhost:11123/api/v1/offline/asr',
        files=files
    )
    
result = response.json()
print(f"识别结果: {result['text']}")
print(f"音频时长: {result['duration']:.2f} 秒")
```

## 测试脚本

使用提供的测试脚本测试不同格式：

```bash
# 测试单个文件
python test_audio_formats.py test.mp3

# 测试多个文件
python test_audio_formats.py audio1.wav audio2.mp3 audio3.m4a
```

## 性能考虑

1. **WAV 格式最快**：直接解析，无需转码
2. **压缩格式稍慢**：需要 FFmpeg 解码
3. **推荐预处理**：对于批量处理，建议预先转换为 WAV 格式

## 质量建议

为了获得最佳识别效果：

1. **采样率**：16kHz 或更高（会自动重采样）
2. **音质**：清晰录音，减少背景噪音
3. **比特率**（压缩格式）：
   - MP3：至少 128kbps
   - AAC/M4A：至少 96kbps
   - OPUS：至少 64kbps
4. **录音设备**：使用质量较好的麦克风

## 限制说明

1. **文件大小**：建议单个文件不超过 100MB
2. **音频时长**：建议不超过 1小时
3. **实时流**：WebSocket 接口仍需要 PCM 格式

## 格式检测

检查当前系统支持的格式：

```bash
curl http://localhost:11123/
```

返回的 JSON 中 `supported_audio_formats` 字段列出所有支持的格式。

示例响应：
```json
{
  "service": "AI Recorder",
  "version": "1.0.0",
  "supported_audio_formats": [
    "wav",
    "mp3",
    "m4a",
    "flac",
    "ogg",
    "opus",
    "aac",
    "wma",
    "amr"
  ]
}
```

如果只返回 `["wav"]`，说明 FFmpeg 未安装。

## 故障排除

### 问题：只支持 WAV 格式

**原因**：FFmpeg 未安装或不在 PATH 中

**解决方法**：
```bash
# 检查 FFmpeg 是否可用
ffmpeg -version

# 如果命令不存在，安装 FFmpeg
# macOS
brew install ffmpeg

# Linux
sudo apt-get install ffmpeg
```

### 问题：转换失败

**可能原因**：
1. 音频文件损坏
2. 不支持的编码格式
3. FFmpeg 版本过旧

**解决方法**：
1. 使用 FFmpeg 手动测试转换：
   ```bash
   ffmpeg -i input.mp3 -ar 16000 -ac 1 output.wav
   ```
2. 更新 FFmpeg 到最新版本
3. 检查服务日志获取详细错误信息

### 问题：识别结果不准确

**可能原因**：
1. 音频质量差
2. 有大量背景噪音
3. 说话不清晰

**解决方法**：
1. 提高录音质量
2. 使用降噪工具预处理：
   ```bash
   ffmpeg -i input.mp3 -af "highpass=f=200, lowpass=f=3000" output.wav
   ```
3. 确保音量适中（不要过大或过小）

## 代码集成

### Go 语言

```go
import (
    "bytes"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

func uploadAudio(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    part, err := writer.CreateFormFile("audio_file", filename)
    if err != nil {
        return err
    }
    
    io.Copy(part, file)
    writer.Close()

    req, err := http.NewRequest("POST", 
        "http://localhost:11123/api/v1/offline/asr", 
        body)
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", writer.FormDataContentType())
    
    client := &http.Client{}
    resp, err := client.Do(req)
    // ... 处理响应
    
    return nil
}
```

### JavaScript/Node.js

```javascript
const FormData = require('form-data');
const fs = require('fs');
const axios = require('axios');

async function uploadAudio(filepath) {
    const form = new FormData();
    form.append('audio_file', fs.createReadStream(filepath));
    
    const response = await axios.post(
        'http://localhost:11123/api/v1/offline/asr',
        form,
        { headers: form.getHeaders() }
    );
    
    console.log('识别结果:', response.data.text);
    console.log('音频时长:', response.data.duration);
}

// 使用
uploadAudio('recording.mp3');
```

## 更多信息

- 完整 API 文档：[API_DOCS.md](../API_DOCS.md)
- 项目主页：[README.md](../README.md)
- 测试脚本：[test_audio_formats.py](../test_audio_formats.py)
