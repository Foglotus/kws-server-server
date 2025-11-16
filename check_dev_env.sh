#!/bin/bash

# 本地开发环境检查脚本

echo "=== AI Recorder 本地开发环境检查 ==="
echo ""

# 检查 Go
echo "1. 检查 Go 环境..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo "   ✓ Go 已安装: $GO_VERSION"
else
    echo "   ✗ Go 未安装"
    echo "   请访问 https://go.dev/dl/ 或运行: brew install go"
    exit 1
fi

# 检查 Go 版本
GO_VERSION_NUM=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
if [ "$(echo "$GO_VERSION_NUM >= 1.21" | bc)" -eq 1 ]; then
    echo "   ✓ Go 版本符合要求 (>= 1.21)"
else
    echo "   ⚠ Go 版本较低，建议升级到 1.21+"
fi

# 检查 GOPATH
echo ""
echo "2. 检查 GOPATH..."
if [ -n "$GOPATH" ]; then
    echo "   ✓ GOPATH: $GOPATH"
else
    echo "   ⚠ GOPATH 未设置，将使用默认值"
fi

# 检查 CGO
echo ""
echo "3. 检查 CGO..."
if [ "$CGO_ENABLED" = "1" ] || [ -z "$CGO_ENABLED" ]; then
    echo "   ✓ CGO 已启用"
else
    echo "   ⚠ CGO 未启用，可能导致某些库无法编译"
fi

# 检查模型文件
echo ""
echo "4. 检查模型文件..."
MODEL_OK=true

if [ ! -f "./models/vad/silero_vad.onnx" ]; then
    echo "   ✗ VAD 模型未找到"
    MODEL_OK=false
else
    echo "   ✓ VAD 模型存在"
fi

if [ ! -d "./models/streaming" ] || [ -z "$(ls -A ./models/streaming 2>/dev/null)" ]; then
    echo "   ✗ 流式识别模型未找到"
    MODEL_OK=false
else
    echo "   ✓ 流式识别模型存在"
fi

if [ ! -d "./models/offline" ] || [ -z "$(ls -A ./models/offline 2>/dev/null)" ]; then
    echo "   ✗ 离线识别模型未找到"
    MODEL_OK=false
else
    echo "   ✓ 离线识别模型存在"
fi

if [ "$MODEL_OK" = false ]; then
    echo ""
    echo "   请运行: ./download_models.sh"
fi

# 检查配置文件
echo ""
echo "5. 检查配置文件..."
if [ -f "./config.yaml" ]; then
    echo "   ✓ config.yaml 存在"
else
    echo "   ✗ config.yaml 不存在"
fi

if [ -f "./config.local.yaml" ]; then
    echo "   ✓ config.local.yaml 存在（本地开发配置）"
else
    echo "   ℹ config.local.yaml 不存在（可选）"
fi

# 检查依赖
echo ""
echo "6. 检查 Go 依赖..."
if [ -f "./go.mod" ]; then
    echo "   ✓ go.mod 存在"
    
    # 尝试验证依赖
    if go mod verify &> /dev/null; then
        echo "   ✓ 依赖完整性验证通过"
    else
        echo "   ⚠ 依赖可能需要更新"
        echo "   运行: go mod download"
    fi
else
    echo "   ✗ go.mod 不存在"
    exit 1
fi

# 检查可选工具
echo ""
echo "7. 检查开发工具..."

if command -v air &> /dev/null; then
    echo "   ✓ air (热重载) 已安装"
else
    echo "   ○ air (热重载) 未安装"
    echo "     安装: go install github.com/cosmtrek/air@latest"
fi

if command -v dlv &> /dev/null; then
    echo "   ✓ delve (调试器) 已安装"
else
    echo "   ○ delve (调试器) 未安装"
    echo "     安装: go install github.com/go-delve/delve/cmd/dlv@latest"
fi

if command -v golangci-lint &> /dev/null; then
    echo "   ✓ golangci-lint (代码检查) 已安装"
else
    echo "   ○ golangci-lint (代码检查) 未安装"
    echo "     安装: brew install golangci-lint"
fi

if command -v goimports &> /dev/null; then
    echo "   ✓ goimports (导入管理) 已安装"
else
    echo "   ○ goimports (导入管理) 未安装"
    echo "     安装: go install golang.org/x/tools/cmd/goimports@latest"
fi

# 检查 Python（用于测试）
echo ""
echo "8. 检查 Python 环境（用于测试）..."
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version)
    echo "   ✓ Python3 已安装: $PYTHON_VERSION"
    
    # 检查测试依赖
    if python3 -c "import requests" &> /dev/null; then
        echo "   ✓ requests 已安装"
    else
        echo "   ○ requests 未安装"
        echo "     安装: pip3 install -r requirements.txt"
    fi
    
    if python3 -c "import websockets" &> /dev/null; then
        echo "   ✓ websockets 已安装"
    else
        echo "   ○ websockets 未安装"
        echo "     安装: pip3 install -r requirements.txt"
    fi
else
    echo "   ○ Python3 未安装（测试脚本可选）"
fi

# 检查端口
echo ""
echo "9. 检查端口占用..."
if lsof -i :11123 &> /dev/null; then
    echo "   ⚠ 端口 11123 已被占用"
    echo "   占用进程:"
    lsof -i :11123 | grep LISTEN
    echo "   请停止占用进程或修改配置文件中的端口"
else
    echo "   ✓ 端口 11123 可用"
fi

# 总结
echo ""
echo "=== 检查完成 ==="
echo ""

if [ "$MODEL_OK" = false ]; then
    echo "⚠ 缺少模型文件，请先运行: ./download_models.sh"
    echo ""
fi

echo "准备开始开发？运行以下命令之一："
echo ""
echo "  # 使用本地配置直接运行"
echo "  make dev"
echo ""
echo "  # 或使用热重载（需要先安装 air）"
echo "  make watch"
echo ""
echo "  # 或手动运行"
echo "  CONFIG_PATH=./config.local.yaml go run main.go"
echo ""
echo "更多信息请查看: DEVELOPMENT.md"
