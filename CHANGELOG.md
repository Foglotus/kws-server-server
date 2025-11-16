# 项目变更日志

所有项目的重要变更都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [未发布]

### 新增
- 版本管理系统
  - 添加 VERSION 文件
  - 版本信息嵌入到二进制文件
  - 构建时注入版本、Git提交和构建时间
  - 命令行版本查询支持 (`-v`, `-version`)
  - API 接口返回版本信息
- 发布管理
  - `make version` - 查看版本信息
  - `make release` - 打包发布版本
  - `deploy.sh` 支持版本参数

## [1.0.0] - 2025-01-13

### 新增功能

- ✨ 实时语音识别 (WebSocket)
  - 支持持续音频流处理
  - 自动端点检测
  - 多会话并发支持

- ✨ 离线语音识别 (HTTP POST)
  - 支持 Base64 编码和文件上传
  - 支持多种 ASR 模型 (Paraformer, Whisper, Transducer)
  - 高并发批处理

- ✨ 说话者分离功能
  - 自动检测和分离多个说话者
  - 与 ASR 无缝集成
  - 支持独立使用

- 🐳 Docker 部署支持
  - ARM64 架构优化
  - 一键部署脚本
  - 健康检查和监控

- 📊 监控和统计
  - 实时会话统计
  - 离线任务统计
  - 服务健康检查

### 技术特性

- 基于 sherpa-onnx 核心库
- Go 语言实现，高性能
- Gin Web 框架
- WebSocket 支持
- 模块化设计
- 配置文件驱动

### 文档

- 📖 完整的 README
- 🚀 快速入门指南
- 📝 API 详细文档
- 🧪 测试脚本和示例

### 支持的平台

- ARM64 (Apple Silicon, Raspberry Pi 4+)
- Linux/macOS
- Docker 容器化部署

---

## 未来计划

### v1.1.0 (计划中)

- [ ] 支持更多 ASR 模型
- [ ] GPU 加速支持
- [ ] 流式说话者分离
- [ ] 多语言支持优化
- [ ] 性能优化

### v1.2.0 (计划中)

- [ ] Web 管理界面
- [ ] 用户认证和授权
- [ ] 速率限制
- [ ] API Key 管理
- [ ] 更详细的监控指标

### v2.0.0 (计划中)

- [ ] 支持 x86_64 架构
- [ ] Kubernetes 部署支持
- [ ] 微服务架构拆分
- [ ] 消息队列集成
- [ ] 分布式部署

---

## 贡献

欢迎提交 Issue 和 Pull Request！

详见 [贡献指南](CONTRIBUTING.md)
