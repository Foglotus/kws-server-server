# 测试指南

本文档介绍如何测试 AI Recorder 语音识别服务。

## 🌐 Web 测试页面（推荐）

### 访问地址

启动服务后，在浏览器中打开：

```
http://localhost:11123/test
```

如果部署在其他服务器上，将 `localhost` 替换为相应的 IP 地址或域名。

### 功能说明

#### 1. 服务状态监控

页面顶部显示服务的实时状态：
- **服务状态**: 显示后端服务是否正常运行
- **WebSocket**: 显示实时识别的连接状态
- **服务器地址**: 当前连接的服务器地址

#### 2. 配置区域

可以自定义以下配置：
- **服务器地址**: 如果服务部署在其他地址，可在此修改
- **采样率**: 音频采样率，默认 16000Hz

点击"更新配置"按钮使配置生效。

#### 3. 实时语音识别

**步骤：**
1. 点击"开始录音"按钮
2. 浏览器会请求麦克风权限，点击"允许"
3. 对着麦克风说话
4. 识别结果会实时显示在结果区域
5. 点击"停止录音"结束识别

**注意事项：**
- 需要使用 HTTPS 或 localhost 才能访问麦克风
- 确保麦克风设备正常工作
- 说话清晰，避免环境噪音

#### 4. 离线语音识别

**步骤：**
1. 点击"选择音频文件"上传音频
2. 支持 WAV, MP3, M4A 等常见格式
3. 点击"上传并识别"
4. 等待处理完成，查看识别结果

**适用场景：**
- 批量处理录音文件
- 处理较长的音频
- 对单个说话者的录音进行识别

#### 5. 说话者分离识别

**步骤：**
1. 点击"选择音频文件"上传包含多人对话的音频
2. 点击"上传并分析"
3. 等待处理完成（可能需要较长时间）
4. 查看按说话者分段的识别结果

**适用场景：**
- 会议录音
- 多人对话场景
- 需要区分不同说话者的场合

**结果展示：**
- 每个说话者用不同颜色标识
- 显示说话者编号（Speaker 0, Speaker 1...）
- 显示时间范围（开始时间 - 结束时间）
- 显示对应的文本内容

#### 6. API 测试

提供三个测试按钮：
- **测试健康检查**: 测试 `/health` 端点
- **获取服务信息**: 测试 `/` 端点，查看可用的 API
- **获取统计信息**: 测试 `/api/v1/stats` 端点

点击按钮后，会在下方显示 JSON 格式的响应数据。

### 常见问题

**Q: 页面显示"无法连接"？**

A: 请检查：
1. 服务是否已启动（`docker-compose ps` 查看）
2. 端口 11123 是否被占用
3. 防火墙是否开放 11123 端口
4. 服务器地址配置是否正确

**Q: 实时识别无法使用麦克风？**

A: 请检查：
1. 浏览器是否已授予麦克风权限
2. 是否使用 HTTPS 或 localhost 访问
3. 麦克风设备是否正常工作
4. 是否有其他程序占用麦克风

**Q: WebSocket 连接失败？**

A: 请检查：
1. 后端服务是否正常运行
2. WebSocket 端点是否正确（`ws://localhost:11123/api/v1/streaming/asr`）
3. 如果使用 HTTPS 访问页面，需要确保 WebSocket 使用 WSS 协议
4. 防火墙或代理是否阻止了 WebSocket 连接

**Q: 文件上传失败？**

A: 请检查：
1. 文件大小是否超过限制（建议小于 100MB）
2. 文件格式是否支持
3. 服务器磁盘空间是否充足
4. 查看浏览器控制台的错误信息

**Q: 识别结果不准确？**

A: 可能的原因：
1. 音频质量较差或有较多噪音
2. 说话不清晰或语速过快
3. 使用的模型不适合当前场景
4. 采样率设置不匹配

## 📝 命令行测试

如果不方便使用浏览器，也可以使用命令行工具测试。

### 1. 健康检查

```bash
curl http://localhost:11123/health
```

预期响应：
```json
{
  "status": "healthy",
  "service": "airecorder"
}
```

### 2. 获取服务信息

```bash
curl http://localhost:11123/
```

### 3. 离线识别（文件上传）

```bash
curl -X POST http://localhost:11123/api/v1/offline/asr \
  -F "audio_file=@/path/to/your/audio.wav"
```

### 4. 说话者分离识别

```bash
curl -X POST http://localhost:11123/api/v1/offline/asr/diarization \
  -F "audio_file=@/path/to/your/audio.wav"
```

### 5. WebSocket 测试（Python）

项目中提供了 Python 测试脚本：

```bash
# 安装依赖
pip install -r requirements.txt

# 运行 WebSocket 测试
python test_websocket.py

# 运行 API 测试
python test_api.py
```

## 🔍 问题排查

### 查看日志

```bash
# Docker 部署
docker-compose logs -f

# 查看最近 100 行日志
docker-compose logs --tail=100

# 只查看错误日志
docker-compose logs | grep -i error
```

### 检查服务状态

```bash
# 查看容器状态
docker-compose ps

# 查看容器资源使用
docker stats airecorder
```

### 进入容器调试

```bash
# 进入容器
docker-compose exec airecorder sh

# 查看模型文件
ls -lh /models/

# 检查配置文件
cat /app/config.yaml
```

### 端口检查

```bash
# 检查端口是否被占用
lsof -i :11123

# 或使用 netstat
netstat -an | grep 11123
```

## 📊 性能测试

### 并发测试

可以使用 Apache Bench (ab) 进行简单的并发测试：

```bash
# 安装 ab
# macOS: brew install apache2
# Ubuntu: apt-get install apache2-utils

# 测试健康检查端点
ab -n 1000 -c 10 http://localhost:11123/health
```

### 压力测试

使用 `wrk` 进行更专业的压力测试：

```bash
# 安装 wrk
# macOS: brew install wrk
# Ubuntu: 需要从源码编译

# 测试
wrk -t4 -c100 -d30s http://localhost:11123/health
```

## 💡 最佳实践

1. **首次使用**: 先用 Web 测试页面快速了解各功能
2. **开发调试**: 使用命令行工具或 Python 脚本
3. **自动化测试**: 集成到 CI/CD 流程中
4. **性能测试**: 在类生产环境中进行压力测试
5. **监控告警**: 定期检查 `/health` 和 `/api/v1/stats` 端点

## 🆘 获取帮助

如果遇到问题：

1. 查看 `DEVELOPMENT.md` 了解开发细节
2. 查看 `API_DOCS.md` 了解 API 详细文档
3. 检查日志文件寻找错误信息
4. 提交 Issue 描述问题（附上日志和配置）

---

**提示**: Web 测试页面会自动检测服务器地址，大多数情况下无需手动配置！
