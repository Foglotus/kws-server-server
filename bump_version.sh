#!/bin/bash

##############################################################################
# 版本更新脚本
# 用途：自动更新版本号并创建标签
##############################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_FILE="$SCRIPT_DIR/VERSION"
CHANGELOG_FILE="$SCRIPT_DIR/CHANGELOG.md"

# 显示用法
show_usage() {
    echo "用法: $0 <major|minor|patch|VERSION>"
    echo ""
    echo "示例:"
    echo "  $0 patch      # 1.0.0 -> 1.0.1"
    echo "  $0 minor      # 1.0.0 -> 1.1.0"
    echo "  $0 major      # 1.0.0 -> 2.0.0"
    echo "  $0 1.2.3      # 直接设置为 1.2.3"
    echo ""
    exit 1
}

# 检查参数
if [ $# -ne 1 ]; then
    show_usage
fi

# 读取当前版本
if [ ! -f "$VERSION_FILE" ]; then
    log_error "VERSION 文件不存在"
    exit 1
fi

CURRENT_VERSION=$(cat "$VERSION_FILE")
log_info "当前版本: $CURRENT_VERSION"

# 解析版本号
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# 计算新版本
case "$1" in
    major)
        NEW_VERSION="$((MAJOR + 1)).0.0"
        ;;
    minor)
        NEW_VERSION="${MAJOR}.$((MINOR + 1)).0"
        ;;
    patch)
        NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
        ;;
    *)
        # 假设是完整版本号
        if [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            NEW_VERSION="$1"
        else
            log_error "无效的版本号格式: $1"
            show_usage
        fi
        ;;
esac

log_info "新版本: $NEW_VERSION"

# 确认
echo ""
read -p "确认更新版本从 $CURRENT_VERSION 到 $NEW_VERSION? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warning "已取消"
    exit 0
fi

# 检查是否有未提交的更改
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    log_warning "检测到未提交的更改"
    git status --short
    echo ""
    read -p "是否继续? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_warning "已取消"
        exit 0
    fi
fi

# 更新 VERSION 文件
echo ""
log_info "更新 VERSION 文件..."
echo "$NEW_VERSION" > "$VERSION_FILE"
log_success "VERSION 文件已更新"

# 更新 CHANGELOG.md
log_info "准备更新 CHANGELOG.md..."
RELEASE_DATE=$(date '+%Y-%m-%d')

# 检查是否有 [未发布] 章节
if grep -q "## \[未发布\]" "$CHANGELOG_FILE"; then
    log_info "将 [未发布] 章节标记为 $NEW_VERSION..."
    
    # 创建临时文件
    TMP_FILE=$(mktemp)
    
    # 替换 [未发布] 为新版本
    sed "s/## \[未发布\]/## [$NEW_VERSION] - $RELEASE_DATE/" "$CHANGELOG_FILE" > "$TMP_FILE"
    
    # 在新版本后添加新的 [未发布] 章节
    awk -v new_version="$NEW_VERSION" -v release_date="$RELEASE_DATE" '
        /^## \['$NEW_VERSION'\]/ {
            print "## [未发布]\n"
            print "### 新增"
            print ""
            print "### 修复"
            print ""
            print "### 变更"
            print ""
            print ""
        }
        { print }
    ' "$TMP_FILE" > "$CHANGELOG_FILE"
    
    rm "$TMP_FILE"
    log_success "CHANGELOG.md 已更新"
else
    log_warning "CHANGELOG.md 中未找到 [未发布] 章节"
    log_info "请手动更新 CHANGELOG.md"
fi

# 提交更改
echo ""
log_info "提交更改..."
git add "$VERSION_FILE" "$CHANGELOG_FILE"
git commit -m "chore: bump version to $NEW_VERSION"
log_success "更改已提交"

# 创建标签
log_info "创建 Git 标签 v$NEW_VERSION..."
git tag -a "v$NEW_VERSION" -m "Release version $NEW_VERSION"
log_success "标签已创建"

# 完成
echo ""
echo "=========================================="
log_success "版本更新完成！"
echo "=========================================="
echo ""
echo "📋 版本信息:"
echo "  - 旧版本: $CURRENT_VERSION"
echo "  - 新版本: $NEW_VERSION"
echo "  - 标签:   v$NEW_VERSION"
echo ""
echo "🚀 下一步:"
echo "  1. 检查更改: git log -1"
echo "  2. 推送代码: git push origin main"
echo "  3. 推送标签: git push origin v$NEW_VERSION"
echo "  4. 构建发布: make release"
echo "  5. 离线部署: cd offline_deploy && ./prepare.sh"
echo ""
echo "📖 详细说明请查看: RELEASE_GUIDE.md"
echo ""
