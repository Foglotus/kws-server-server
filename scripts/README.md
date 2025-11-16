# AI Recorder 离线部署包

## 📦 快速部署

### 一键部署（推荐）

```bash
./deploy.sh
```

部署脚本会自动完成：
- ✅ 加载 Docker 镜像
- ✅ 解压模型文件到 `/opt/airecorder/models`
- ✅ 启动服务容器
- ✅ 健康检查

### 验证部署

```bash
# 验证文件完整性
./verify.sh

# 测试环境
./test_env.sh

# 查看服务状态
docker ps | grep airecorder

# 健康检查
curl http://localhost:11123/health
```

## 📋 系统要求

- **操作系统**: Linux（支持 Docker）
- **CPU 架构**: ARM64
- **Docker**: 20.10+
- **内存**: 最小 4GB RAM
- **磁盘**: 最小 10GB 可用空间

## 🌐 访问服务

部署完成后访问：

- **服务首页**: http://localhost:11123
- **健康检查**: http://localhost:11123/health
- **API 文档**: 查看服务首页

## 🔧 管理命令

```bash
# 查看日志
docker logs -f airecorder

# 重启服务
docker restart airecorder

# 停止服务
docker stop airecorder

# 启动服务
docker start airecorder

# 查看版本
docker exec airecorder ./airecorder -v
```

## 📝 配置说明

服务安装在 `/opt/airecorder/`：

```
/opt/airecorder/
├── models/          # AI 模型文件
├── logs/           # 日志目录
└── config.yaml     # 配置文件
```

如需修改配置，编辑 `/opt/airecorder/config.yaml` 后重启服务。

## 🔍 故障排查

### 服务无法启动？

```bash
# 查看容器日志
docker logs airecorder

# 检查容器状态
docker ps -a | grep airecorder

# 检查磁盘空间
df -h
```

### 端口被占用？

```bash
# 查看端口占用
netstat -tunlp | grep 11123

# 或修改端口，停止容器后重新运行：
docker stop airecorder && docker rm airecorder
docker run -d --name airecorder \
  -p 8080:11123 \
  -v /opt/airecorder/models:/models:ro \
  airecorder:latest
```

## 📖 更多信息

查看 MANIFEST.txt 了解版本和文件清单。
