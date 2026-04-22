#!/bin/bash
# 运行AI Recorder的单元测试
# 使用方法: ./run_tests.sh [test_pattern]

set -e

# 设置项目根目录
PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_ROOT"

# 设置配置文件路径
export CONFIG_PATH="$PROJECT_ROOT/config.test.yaml"

# 设置项目根目录环境变量（测试中使用）
export PROJECT_ROOT="$PROJECT_ROOT"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}  AI Recorder 单元测试${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""
echo -e "配置文件: ${YELLOW}$CONFIG_PATH${NC}"
echo -e "项目根目录: ${YELLOW}$PROJECT_ROOT${NC}"
echo ""

# 如果提供了测试模式参数
if [ -n "$1" ]; then
    echo -e "${YELLOW}运行测试模式:${NC} $1"
    go test -v ./internal/asr ./internal/handler -run "$1" -timeout 300s
else
    echo -e "${YELLOW}运行所有测试...${NC}"
    echo ""
    
    # 运行测试并捕获结果
    if go test -v ./internal/asr ./internal/handler -timeout 300s; then
        echo ""
        echo -e "${GREEN}=====================================${NC}"
        echo -e "${GREEN}  所有测试通过! ✓${NC}"
        echo -e "${GREEN}=====================================${NC}"
        exit 0
    else
        echo ""
        echo -e "${RED}=====================================${NC}"
        echo -e "${RED}  部分测试失败! ✗${NC}"
        echo -e "${RED}=====================================${NC}"
        exit 1
    fi
fi
