#!/bin/bash

##############################################################################
# AI Recorder 离线部署 - 快速测试脚本
# 用于在准备阶段验证所有依赖是否就绪
##############################################################################

echo "=========================================="
echo "  AI Recorder 离线部署 - 环境检查"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0

# 检查 Docker
echo -n "检查 Docker: "
if command -v docker &> /dev/null; then
    VERSION=$(docker --version)
    echo -e "${GREEN}✓${NC} $VERSION"
else
    echo -e "${RED}✗ 未安装${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 检查 Docker 服务
echo -n "检查 Docker 服务: "
if docker info &> /dev/null; then
    echo -e "${GREEN}✓ 运行中${NC}"
else
    echo -e "${RED}✗ 未运行${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 检查磁盘空间
echo -n "检查磁盘空间: "
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AVAILABLE=$(df -BG "$SCRIPT_DIR" | tail -1 | awk '{print $4}' | sed 's/G//')
if [ "$AVAILABLE" -gt 5 ]; then
    echo -e "${GREEN}✓ ${AVAILABLE}GB 可用${NC}"
else
    echo -e "${RED}✗ 仅 ${AVAILABLE}GB 可用 (需要至少 5GB)${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 检查网络连接（尝试访问 GitHub）
echo -n "检查网络连接: "
if curl -s --max-time 5 https://github.com &> /dev/null; then
    echo -e "${GREEN}✓ 正常${NC}"
else
    echo -e "${YELLOW}⚠ 无法访问 GitHub (可能会影响模型下载)${NC}"
fi

# 检查项目文件
echo -n "检查项目结构: "
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
if [ -f "$PROJECT_ROOT/Dockerfile" ] && [ -f "$PROJECT_ROOT/download_models.sh" ]; then
    echo -e "${GREEN}✓ 完整${NC}"
else
    echo -e "${RED}✗ 不完整${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 总结
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ 环境检查通过，可以执行 ./prepare.sh${NC}"
    exit 0
else
    echo -e "${RED}✗ 发现 $ERRORS 个问题，请先解决后再继续${NC}"
    exit 1
fi
