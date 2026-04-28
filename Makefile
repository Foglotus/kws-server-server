.PHONY: help setup download-models build build-native build-binary build-base up down restart logs clean test test-websocket run version release-runtime quick-update

# 版本信息
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X 'airecorder/internal/version.Version=$(VERSION)' \
           -X 'airecorder/internal/version.GitCommit=$(GIT_COMMIT)' \
           -X 'airecorder/internal/version.BuildTime=$(BUILD_TIME)'

# Docker 命令检测
DOCKER := $(shell if [ -f /usr/local/bin/docker ]; then echo /usr/local/bin/docker; elif command -v docker &> /dev/null; then echo docker; fi)
DOCKER_PATH := /Applications/Docker.app/Contents/Resources/bin

# 默认目标
help:
	@echo "=========================================="
	@echo "  AI Recorder - 可用命令"
	@echo "=========================================="
	@echo ""
	@echo "📦 发布打包:"
	@echo "  make release-runtime - ⚡ 运行时部署包"
	@echo "                         分离模式：基础镜像+程序，支持快速更新"
	@echo "  make build-binary    - 🚀 编译程序（快速更新）"
	@echo "                         仅编译二进制，仅几MB"
	@echo "  make quick-update    - ⚡ 快速更新部署"
	@echo "                         仅更新程序，无需传输镜像"
	@echo ""
	@echo "🏗️  基础设施:"
	@echo "  make build-base      - 构建基础运行环境镜像（一次性）"
	@echo "  make download-models - 下载 AI 模型文件"
	@echo ""
	@echo "🚀 快速开始:"
	@echo "  make build-native    - 本地编译二进制文件"
	@echo "  make run             - 运行本地编译的服务"
	@echo ""
	@echo "🐳 Docker 操作:"
	@echo "  make build           - 构建 Docker 镜像"
	@echo "  make up              - 启动 Docker 服务"
	@echo "  make down            - 停止 Docker 服务"
	@echo "  make restart         - 重启 Docker 服务"
	@echo "  make logs            - 查看 Docker 日志"
	@echo ""
	@echo "🔧 开发工具:"
	@echo "  make version         - 显示版本信息"
	@echo "  make test            - 运行健康检查"
	@echo "  make test-go         - 运行 Go 单元测试"
	@echo "  make clean           - 清理容器、镜像和所有生成文件"
	@echo "  make clean-release   - 清理发布文件"
	@echo ""
	@echo "📖 发布流程:"
	@echo ""
	@echo "  【运行时部署】（约2GB首次，后续15MB）"
	@echo "  1. make download-models  # 下载模型"
	@echo "  2. make release-runtime  # 生成运行时部署包"
	@echo "     首次: 传输完整包（基础镜像+程序+模型）"
	@echo "     更新: 仅传输 bin/ 目录，10-20倍速度提升"
	@echo ""
	@echo "  【快速更新】（仅几MB）"
	@echo "  1. make build-binary     # 编译新程序"
	@echo "  2. make quick-update     # 打包并更新"
	@echo "     或手动: scp -r bin/ user@server:/path/"
	@echo ""
	@echo "详细说明: RELEASE_SIMPLE.md"
	@echo "=========================================="
	@echo ""

# 完整设置
setup: download-models build-native

# 下载模型
download-models:
	@echo "下载模型文件..."
	@chmod +x download_models.sh
	@./download_models.sh
	@chmod +x update_config.sh
	@./update_config.sh

# 本地编译
build-native:
	@echo "本地编译 Go 程序 (版本: $(VERSION))..."
	@go mod download
	@go build -ldflags "$(LDFLAGS)" -o airecorder .
	@echo "编译完成! 二进制文件: ./airecorder"
	@./airecorder -v

# 运行本地编译的服务
run:
	@echo "启动本地服务..."
	@if [ ! -f ./airecorder ]; then \
		echo "错误: 二进制文件不存在，请先运行 make build-native"; \
		exit 1; \
	fi
	@./airecorder

# 构建 Docker 镜像
build:
	@echo "构建 Docker 镜像 (版本: $(VERSION))..."
	@if [ -z "$(DOCKER)" ]; then \
		echo "错误: Docker 未安装，请使用 'make build-native' 进行本地编译"; \
		exit 1; \
	fi
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME)
	@echo "打标签: airecorder:$(VERSION)"
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) tag airecorder:latest airecorder:$(VERSION)

# 启动服务
up:
	@echo "启动服务..."
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose up -d
	@echo "等待服务启动..."
	@sleep 10
	@make status

# 停止服务
down:
	@echo "停止服务..."
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose down

# 重启服务
restart:
	@echo "重启服务..."
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose restart
	@sleep 5
	@make status

# 查看日志
logs:
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose logs -f

# 查看服务状态
status:
	@echo "服务状态:"
	@PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose ps
	@echo ""
	@echo "健康检查:"
	@curl -s http://localhost:11123/health || echo "服务未响应"

# 运行健康检查
test:
	@echo "运行健康检查..."
	@curl -s http://localhost:11123/health | grep -q "healthy" && echo "✓ 服务健康" || echo "✗ 服务异常"
	@echo ""
	@echo "获取服务信息..."
	@curl -s http://localhost:11123/ | python3 -m json.tool || echo "✗ 获取失败"

# 运行 API 测试
test-api:
	@echo "运行 API 测试..."
	@if [ -z "$(AUDIO)" ]; then \
		echo "错误: 请指定音频文件"; \
		echo "用法: make test-api AUDIO=test.wav"; \
		exit 1; \
	fi
	@python3 test_api.py --audio $(AUDIO)

# 运行 WebSocket 测试
test-websocket:
	@echo "运行 WebSocket 测试..."
	@if [ -z "$(AUDIO)" ]; then \
		echo "错误: 请指定音频文件"; \
		echo "用法: make test-websocket AUDIO=test.wav"; \
		exit 1; \
	fi
	@python3 test_websocket.py --audio $(AUDIO)

# 查看统计信息
stats:
	@echo "服务统计信息:"
	@curl -s http://localhost:11123/api/v1/stats | python3 -m json.tool

# 清理容器和镜像
clean:
	@echo "清理 Docker 容器和镜像..."
	@if [ -n "$(DOCKER)" ]; then \
		PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose down -v; \
		PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose -f docker-compose.runtime.yml down -v 2>/dev/null || true; \
		PATH=$(DOCKER_PATH):$$PATH $(DOCKER) rmi airecorder:latest 2>/dev/null || true; \
		PATH=$(DOCKER_PATH):$$PATH $(DOCKER) rmi airecorder-base:latest 2>/dev/null || true; \
	fi
	@echo "清理本地二进制文件..."
	@rm -f airecorder
	@rm -rf bin/
	@echo "清理发布文件..."
	@rm -f offline_deploy/airecorder.tar.gz
	@rm -f offline_deploy/airecorder-base.tar.gz
	@rm -f offline_deploy/models.tar.gz
	@rm -f offline_deploy/VERSION
	@rm -f offline_deploy/checksums.md5
	@rm -f offline_deploy/MANIFEST.txt
	@rm -rf offline_deploy/bin/
	@rm -rf offline_deploy/static/
	@rm -f offline_deploy/config.yaml
	@rm -f offline_deploy/*.sh
	@rm -f offline_deploy/README.md
	@echo "✓ 清理完成"

# 完全清理（包括模型）
clean-all: clean
	@echo "清理模型文件..."
	rm -rf models/
	@echo "清理日志文件..."
	rm -rf logs/
	@echo "✓ 完全清理完成"

# 开发模式（本地运行）
dev:
	@echo "开发模式启动..."
	@echo "使用本地配置: config.local.yaml"
	CONFIG_PATH=./config.local.yaml go run main.go

# 使用热重载开发（需要安装 air）
watch:
	@echo "启动热重载开发模式..."
	@which air > /dev/null || (echo "请先安装 air: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# 编译本地版本
build-local:
	@echo "编译本地版本..."
	go build -o airecorder main.go
	@echo "✓ 编译完成: ./airecorder"

# 运行 Go 测试
test-go:
	@echo "运行 Go 单元测试..."
	go test -v ./...

# 格式化代码
fmt:
	@echo "格式化 Go 代码..."
	go fmt ./...
	goimports -w .

# 代码检查
lint:
	@echo "运行代码检查..."
	golangci-lint run

# 安装 Python 依赖
install-deps:
	@echo "安装 Python 测试依赖..."
	pip3 install -r requirements.txt

# 更新配置
update-config:
	@echo "更新配置文件..."
	@chmod +x update_config.sh
	@./update_config.sh

# 查看帮助信息
info:
	@echo "AI Recorder 服务信息"
	@echo "===================="
	@echo ""
	@echo "服务地址: http://localhost:11123"
	@echo ""
	@echo "可用端点:"
	@echo "  - GET  /health                        - 健康检查"
	@echo "  - GET  /                              - 服务信息"
	@echo "  - WS   /api/v1/streaming/asr          - 实时语音识别"
	@echo "  - POST /api/v1/offline/asr            - 离线语音识别"
	@echo "  - POST /api/v1/offline/asr/diarization - 带说话者分离的识别"
	@echo "  - POST /api/v1/diarization            - 独立说话者分离"
	@echo "  - GET  /api/v1/stats                  - 统计信息"
	@echo ""
	@echo "文档:"
	@echo "  - README.md      - 完整文档"
	@echo "  - QUICKSTART.md  - 快速入门"
	@echo "  - API_DOCS.md    - API 详细文档"
	@echo ""

# 显示版本信息
version:
	@echo "AI Recorder 版本信息"
	@echo "===================="
	@echo "版本号:    $(VERSION)"
	@echo "Git提交:   $(GIT_COMMIT)"
	@echo "构建时间:  $(BUILD_TIME)"
	@echo ""
	@if [ -f ./airecorder ]; then \
		echo "已编译的二进制文件版本:"; \
		./airecorder -v 2>/dev/null || echo "  无法运行（可能缺少模型文件）"; \
	else \
		echo "提示: 运行 'make build-native' 编译二进制文件"; \
	fi

# 打包运行时部署包（基础镜像 + 编译程序 + 模型）
release-runtime:
	@echo "=========================================="
	@echo "  AI Recorder 运行时部署包生成"
	@echo "  版本: $(VERSION)"
	@echo "  模式: 基础镜像 + 编译程序"
	@echo "=========================================="
	@echo ""
	
	@echo "步骤 1/6: 检查模型文件..."
	@if [ ! -f "./models/vad/silero_vad.onnx" ]; then \
		echo "❌ 模型文件缺失，请先运行: make download-models"; \
		exit 1; \
	fi
	@echo "✓ 模型文件检查通过"
	@echo ""
	
	@echo "步骤 2/6: 构建基础镜像..."
	@if [ -z "$(DOCKER)" ]; then \
		echo "❌ Docker 未安装或未找到"; \
		exit 1; \
	fi
	@if ! PATH=$(DOCKER_PATH):$$PATH $(DOCKER) images | grep -q airecorder-base; then \
		echo "构建基础运行环境镜像..."; \
		PATH=$(DOCKER_PATH):$$PATH $(DOCKER) build -t airecorder-base:latest -f Dockerfile.base .; \
	else \
		echo "✓ 基础镜像已存在"; \
	fi
	@echo ""
	
	@echo "步骤 3/6: 编译 ARM64 程序..."
	@if [ ! -f "./build_binary.sh" ]; then \
		echo "❌ build_binary.sh 不存在"; \
		exit 1; \
	fi
	@chmod +x ./build_binary.sh
	@./build_binary.sh
	@echo ""
	
	@echo "步骤 4/6: 准备部署目录..."
	@mkdir -p offline_deploy/bin/lib
	@mkdir -p offline_deploy/static
	@cp -f bin/airecorder offline_deploy/bin/
	@cp -rf bin/lib/* offline_deploy/bin/lib/ 2>/dev/null || true
	@cp -rf static/* offline_deploy/static/ 2>/dev/null || true
	@cp config.yaml offline_deploy/ 2>/dev/null || true
	@cp docker-compose.runtime.yml offline_deploy/ 2>/dev/null || true
	@if [ -d scripts ]; then \
		cp scripts/deploy-smart.sh offline_deploy/deploy.sh 2>/dev/null || true; \
		cp scripts/verify.sh offline_deploy/ 2>/dev/null || true; \
		cp scripts/test_env.sh offline_deploy/ 2>/dev/null || true; \
		cp scripts/README.md offline_deploy/ 2>/dev/null || true; \
		chmod +x offline_deploy/*.sh 2>/dev/null || true; \
	fi
	@echo "✓ 文件复制完成"
	@echo ""
	
	@echo "步骤 5/6: 打包模型和镜像..."
	@echo "  打包模型文件..."
	@if [ -d models ] && [ -n "$$(ls -A models 2>/dev/null)" ]; then \
		tar -czf offline_deploy/models.tar.gz models/ && \
		MODEL_SIZE=$$(du -h offline_deploy/models.tar.gz | cut -f1) && \
		echo "  ✓ 模型: $$MODEL_SIZE"; \
	fi
	@echo "  导出基础镜像..."
	@PATH=$(DOCKER_PATH):$$PATH $(DOCKER) save airecorder-base:latest | gzip > offline_deploy/airecorder-base.tar.gz
	@BASE_IMAGE_SIZE=$$(du -h offline_deploy/airecorder-base.tar.gz | cut -f1); \
	echo "  ✓ 基础镜像: $$BASE_IMAGE_SIZE"
	@echo ""
	
	@echo "步骤 6/6: 生成部署清单..."
	@echo "$(VERSION)" > offline_deploy/VERSION
	@BINARY_SIZE=$$(du -h offline_deploy/bin/airecorder | cut -f1); \
	MODEL_SIZE=$$(du -h offline_deploy/models.tar.gz 2>/dev/null | cut -f1 || echo "N/A"); \
	BASE_IMAGE_SIZE=$$(du -h offline_deploy/airecorder-base.tar.gz | cut -f1); \
	echo "AI Recorder 运行时部署包" > offline_deploy/MANIFEST.txt; \
	echo "========================================" >> offline_deploy/MANIFEST.txt; \
	echo "版本号: $(VERSION)" >> offline_deploy/MANIFEST.txt; \
	echo "部署模式: 运行时模式（快速更新）" >> offline_deploy/MANIFEST.txt; \
	echo "构建时间: $(BUILD_TIME)" >> offline_deploy/MANIFEST.txt; \
	echo "" >> offline_deploy/MANIFEST.txt; \
	echo "文件列表:" >> offline_deploy/MANIFEST.txt; \
	echo "----------------------------------------" >> offline_deploy/MANIFEST.txt; \
	echo "1. bin/airecorder             - 编译程序 ($$BINARY_SIZE)" >> offline_deploy/MANIFEST.txt; \
	echo "2. bin/lib/                   - 共享库文件" >> offline_deploy/MANIFEST.txt; \
	echo "3. airecorder-base.tar.gz     - 基础镜像 ($$BASE_IMAGE_SIZE)" >> offline_deploy/MANIFEST.txt; \
	echo "4. models.tar.gz              - AI 模型 ($$MODEL_SIZE)" >> offline_deploy/MANIFEST.txt; \
	echo "5. docker-compose.runtime.yml - 运行时配置" >> offline_deploy/MANIFEST.txt; \
	echo "6. config.yaml                - 配置文件" >> offline_deploy/MANIFEST.txt; \
	echo "7. deploy.sh                  - 智能部署脚本 ⭐" >> offline_deploy/MANIFEST.txt; \
	echo "" >> offline_deploy/MANIFEST.txt; \
	echo "部署方法:" >> offline_deploy/MANIFEST.txt; \
	echo "----------------------------------------" >> offline_deploy/MANIFEST.txt; \
	echo "1. 将 offline_deploy 目录复制到目标机器" >> offline_deploy/MANIFEST.txt; \
	echo "2. cd offline_deploy && chmod +x deploy.sh" >> offline_deploy/MANIFEST.txt; \
	echo "3. ./deploy.sh  # 自动检测并部署" >> offline_deploy/MANIFEST.txt; \
	echo "" >> offline_deploy/MANIFEST.txt; \
	echo "快速更新（仅替换程序）:" >> offline_deploy/MANIFEST.txt; \
	echo "----------------------------------------" >> offline_deploy/MANIFEST.txt; \
	echo "下次代码修改后，只需传输 bin/ 目录：" >> offline_deploy/MANIFEST.txt; \
	echo "  scp -r bin/ user@server:/path/offline_deploy/" >> offline_deploy/MANIFEST.txt; \
	echo "  ssh user@server 'cd /path/offline_deploy && docker-compose -f docker-compose.runtime.yml restart'" >> offline_deploy/MANIFEST.txt; \
	echo "========================================" >> offline_deploy/MANIFEST.txt
	@echo "✓ 清单生成完成"
	@echo ""
	
	@echo "=========================================="
	@echo "✓ 运行时部署包生成完成！"
	@echo "=========================================="
	@echo ""
	@echo "📦 部署包位置: offline_deploy/"
	@echo ""
	@echo "📂 包含文件:"
	@ls -lh offline_deploy/ | tail -n +2 | awk '{print $$9, "-", $$5}'
	@echo ""
	@TOTAL_SIZE=$$(du -sh offline_deploy/ | cut -f1); \
	echo "📊 总大小: $$TOTAL_SIZE"
	@echo ""
	@echo "🚀 打包发布:"
	@echo "   tar -czf airecorder-$(VERSION)-runtime.tar.gz offline_deploy/"
	@echo ""
	@echo "💡 优势:"
	@echo "   - 首次部署: 传输完整包"
	@echo "   - 快速更新: 仅传输 bin/ 目录 (~15MB)"
	@echo "   - 更新速度: 比完整部署快 10-20 倍"
	@echo ""

# 构建基础运行环境镜像（一次性操作）
build-base:
	@echo "=========================================="
	@echo "  构建基础运行环境镜像"
	@echo "=========================================="
	@echo ""
	@if [ -z "$(DOCKER)" ]; then \
		echo "❌ Docker 未安装或未找到"; \
		exit 1; \
	fi
	@echo "构建基础镜像 airecorder-base:latest ..."
	@PATH=$(DOCKER_PATH):$$PATH $(DOCKER) build \
		-t airecorder-base:latest \
		-f Dockerfile.base .
	@echo ""
	@echo "✓ 基础镜像构建完成！"
	@echo "  镜像名称: airecorder-base:latest"
	@echo "  说明: 此镜像只包含运行时依赖，不包含程序"
	@echo "  用途: 配合 docker-compose.runtime.yml 使用"
	@echo ""

# 编译 ARM64 二进制文件（用于快速更新）
build-binary:
	@echo "=========================================="
	@echo "  编译 ARM64 二进制文件"
	@echo "  版本: $(VERSION)"
	@echo "=========================================="
	@echo ""
	@if [ ! -f "./build_binary.sh" ]; then \
		echo "❌ build_binary.sh 不存在"; \
		exit 1; \
	fi
	@chmod +x ./build_binary.sh
	@./build_binary.sh
	@echo ""
	@echo "✓ 编译完成！"
	@echo "  二进制文件: ./bin/airecorder"
	@echo "  共享库: ./bin/lib/"
	@echo ""
	@echo "现在可以使用以下方式更新远程服务器："
	@echo "  1. make quick-update           # 使用快速更新脚本"
	@echo "  2. scp bin/airecorder user@host:/path/bin/"
	@echo ""

# 快速更新（仅更新程序文件）
quick-update:
	@echo "=========================================="
	@echo "  快速更新部署"
	@echo "  版本: $(VERSION)"
	@echo "=========================================="
	@echo ""
	@if [ ! -f "./bin/airecorder" ]; then \
		echo "❌ 二进制文件不存在，请先运行: make build-binary"; \
		exit 1; \
	fi
	@if [ ! -f "./quick_update.sh" ]; then \
		echo "❌ quick_update.sh 不存在"; \
		exit 1; \
	fi
	@chmod +x ./quick_update.sh
	@./quick_update.sh
	@echo ""

# 使用运行时配置启动服务（基础镜像 + 挂载程序）
up-runtime:
	@echo "使用运行时配置启动服务..."
	@if [ ! -f "./bin/airecorder" ]; then \
		echo "❌ 二进制文件不存在"; \
		echo "请先运行: make build-binary"; \
		exit 1; \
	fi
	@if ! PATH=$(DOCKER_PATH):$$PATH $(DOCKER) images | grep -q airecorder-base; then \
		echo "❌ 基础镜像不存在"; \
		echo "请先运行: make build-base"; \
		exit 1; \
	fi
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose -f docker-compose.runtime.yml up -d
	@echo "等待服务启动..."
	@sleep 10
	@echo ""
	@echo "服务状态:"
	@PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose -f docker-compose.runtime.yml ps
	@echo ""
	@echo "健康检查:"
	@curl -s http://localhost:11123/health || echo "服务未响应"

# 停止运行时服务
down-runtime:
	@echo "停止运行时服务..."
	PATH=$(DOCKER_PATH):$$PATH $(DOCKER) compose -f docker-compose.runtime.yml down
	@echo ""

# 清理发布文件
clean-release:
	@echo "清理发布文件..."
	@rm -f offline_deploy/airecorder.tar.gz
	@rm -f offline_deploy/airecorder-base.tar.gz
	@rm -f offline_deploy/models.tar.gz
	@rm -f offline_deploy/VERSION
	@rm -f offline_deploy/checksums.md5
	@rm -f offline_deploy/MANIFEST.txt
	@rm -rf offline_deploy/bin/
	@rm -rf offline_deploy/static/
	@rm -f offline_deploy/config.yaml
	@rm -f offline_deploy/docker-compose.runtime.yml
	@rm -f offline_deploy/*.sh
	@rm -f offline_deploy/README.md
	@echo "✓ 发布文件已清理"
