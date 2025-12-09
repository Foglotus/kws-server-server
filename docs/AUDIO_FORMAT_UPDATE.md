# 音频格式支持功能更新说明

## 更新概述

为 airecorder 语音识别服务添加了多种常见音频格式的自动检测和转换支持。用户现在可以直接上传 MP3、M4A、FLAC、OGG 等格式的音频文件，无需手动转换为 WAV 格式。

## 新增功能

### 1. 自动格式检测
- 服务会自动检测上传音频文件的格式
- 支持基于文件头魔数的格式识别
- 无需用户指定格式类型

### 2. 支持的音频格式

#### 原生支持（无需外部依赖）
- **WAV**: 支持 8/16/32-bit PCM，任意采样率，单/多声道

#### FFmpeg 支持（需安装 FFmpeg）
- **MP3**: MPEG Audio Layer 3
- **M4A/MP4**: MPEG-4 Audio (iPhone 录音常用格式)
- **FLAC**: 无损音频格式
- **OGG**: Ogg Vorbis
- **OPUS**: Opus 编码
- **AAC**: Advanced Audio Coding
- **WMA**: Windows Media Audio
- **AMR**: Adaptive Multi-Rate (手机录音常用)

### 3. 自动音频处理
- 自动重采样至 16kHz（ASR 最佳采样率）
- 自动转换为单声道（如果是多声道）
- 自动转换为 float32 样本数组

## 技术实现

### 新增文件

1. **internal/audio/converter.go**
   - 音频格式检测
   - WAV 格式解析器
   - FFmpeg 集成
   - 采样率转换
   - 声道转换

2. **docs/AUDIO_FORMATS.md**
   - 详细的格式支持说明
   - 安装指南
   - 使用示例
   - 故障排除

3. **test_audio_formats.py**
   - 格式测试脚本
   - 支持批量测试
   - 文件上传和 Base64 两种方式

4. **check_audio_support.sh**
   - 环境检查脚本
   - 自动生成测试文件
   - 快速验证功能

### 修改的文件

1. **internal/handler/asr.go**
   - 使用音频转换器处理上传的文件
   - 支持多种格式的 Base64 编码数据
   - 改进错误处理

2. **internal/handler/health.go**
   - 在服务信息中显示支持的格式列表

3. **API_DOCS.md**
   - 更新音频格式要求部分
   - 添加格式支持说明
   - 添加 FFmpeg 安装指南

4. **README.md**
   - 在功能特性中强调多格式支持

## 使用方法

### 基本使用

```bash
# 直接上传任意支持的格式
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@recording.mp3"

curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@voice_memo.m4a"

curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@audio.flac"
```

### Python 示例

```python
import requests

# 上传 MP3 文件
with open('recording.mp3', 'rb') as f:
    files = {'audio_file': f}
    response = requests.post(
        'http://localhost:11123/api/v1/offline/asr',
        files=files
    )

result = response.json()
print(f"识别结果: {result['text']}")
```

### 检查支持的格式

```bash
# 查看当前支持的格式
curl http://localhost:11123/

# 运行环境检查脚本
./check_audio_support.sh

# 测试多种格式
python test_audio_formats.py test1.wav test2.mp3 test3.m4a
```

## 环境要求

### 必需
- Go 1.21+
- 基础的 WAV 格式支持无需额外依赖

### 可选（推荐）
- **FFmpeg**: 用于支持 MP3、M4A 等压缩格式
  ```bash
  # macOS
  brew install ffmpeg
  
  # Ubuntu/Debian
  sudo apt-get install ffmpeg
  
  # CentOS/RHEL
  sudo yum install ffmpeg
  ```

## 性能考虑

1. **WAV 格式性能最佳**: 无需转码，直接解析
2. **压缩格式略慢**: 需要 FFmpeg 解码，增加约 10-30% 处理时间
3. **推荐批量处理**: 对于大量文件，建议预先转换为 WAV

## 兼容性

### 向后兼容
- 所有现有的 API 接口保持不变
- 原有的 WAV/PCM 处理逻辑完全兼容
- 不影响实时语音识别（WebSocket）接口

### 降级支持
- 如果 FFmpeg 不可用，自动降级为仅支持 WAV
- 服务会在启动时检测 FFmpeg 可用性
- 通过 API 可查询当前支持的格式列表

## 测试

### 单元测试

```bash
# 测试音频转换器
go test ./internal/audio/...

# 测试 handler
go test ./internal/handler/...
```

### 集成测试

```bash
# 启动服务
./bin/airecorder

# 在另一个终端运行测试
python test_audio_formats.py test_*.{wav,mp3,m4a,flac}
```

## 已知限制

1. **实时流**: WebSocket 接口仍需要 PCM 格式
2. **文件大小**: 建议单文件不超过 100MB
3. **格式检测**: 依赖文件头魔数，损坏的文件可能检测失败
4. **采样率**: 所有音频都会重采样至 16kHz

## 故障排除

### 问题：只显示支持 WAV 格式

**原因**: FFmpeg 未安装

**解决**: 
```bash
# 检查 FFmpeg
ffmpeg -version

# 安装 FFmpeg（如果未安装）
brew install ffmpeg  # macOS
```

### 问题：格式转换失败

**原因**: 
- 音频文件损坏
- FFmpeg 版本过旧
- 不支持的编码格式

**解决**:
1. 手动测试 FFmpeg 转换: `ffmpeg -i input.mp3 -ar 16000 output.wav`
2. 更新 FFmpeg: `brew upgrade ffmpeg`
3. 检查服务日志获取详细错误

### 问题：识别结果不准确

**原因**: 音频质量问题

**解决**:
1. 确保音频清晰，无大量噪音
2. 使用降噪工具预处理
3. 提高录音质量

## 未来改进

- [ ] 添加音频质量检测和警告
- [ ] 支持更多压缩格式（如 APE、WV 等）
- [ ] 添加音频预处理选项（降噪、增强等）
- [ ] 优化大文件处理性能
- [ ] 添加音频格式转换缓存

## 相关文档

- [API 文档](../API_DOCS.md) - 完整的 API 接口文档
- [音频格式说明](AUDIO_FORMATS.md) - 详细的格式支持说明
- [README](../README.md) - 项目主文档

## 贡献者

- 开发: @ikd_elaiza
- 日期: 2025-11-28
- 版本: 1.1.0

## 变更日志

### v1.1.0 (2025-11-28)
- ✨ 新增多种音频格式支持（MP3、M4A、FLAC、OGG 等）
- ✨ 新增自动格式检测功能
- ✨ 新增 FFmpeg 集成
- 📝 更新 API 文档
- 📝 新增音频格式文档
- 🧪 新增格式测试脚本
- 🔧 改进错误处理和日志

### v1.0.0 (之前)
- 基础 WAV 格式支持
- 实时和离线语音识别
- 说话者分离功能
