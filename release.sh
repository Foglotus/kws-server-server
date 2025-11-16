#!/bin/bash

# 版本发布助手脚本
# 用于简化版本发布流程

set -e

CURRENT_VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")

echo "=== AI Recorder 版本发布助手 ==="
echo ""
echo "当前版本: $CURRENT_VERSION"
echo ""

# 显示菜单
echo "请选择操作:"
echo "1) 发布补丁版本 (PATCH: 修复bug)"
echo "2) 发布次版本 (MINOR: 新增功能)"
echo "3) 发布主版本 (MAJOR: 重大变更)"
echo "4) 自定义版本号"
echo "5) 退出"
echo ""

read -p "请输入选择 (1-5): " choice

case $choice in
    1)
        # 补丁版本 x.x.X+1
        NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{$NF = $NF + 1;} 1' | sed 's/ /./g')
        VERSION_TYPE="补丁版本 (PATCH)"
        ;;
    2)
        # 次版本 x.X+1.0
        NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{$(NF-1) = $(NF-1) + 1; $NF = 0;} 1' | sed 's/ /./g')
        VERSION_TYPE="次版本 (MINOR)"
        ;;
    3)
        # 主版本 X+1.0.0
        NEW_VERSION=$(echo $CURRENT_VERSION | awk -F. '{$1 = $1 + 1; $2 = 0; $3 = 0;} 1' | sed 's/ /./g')
        VERSION_TYPE="主版本 (MAJOR)"
        ;;
    4)
        read -p "请输入新版本号 (如 1.2.0): " NEW_VERSION
        VERSION_TYPE="自定义版本"
        ;;
    5)
        echo "退出"
        exit 0
        ;;
    *)
        echo "无效选择"
        exit 1
        ;;
esac

echo ""
echo "=== 版本升级 ==="
echo "类型: $VERSION_TYPE"
echo "当前版本: $CURRENT_VERSION"
echo "新版本:   $NEW_VERSION"
echo ""

read -p "是否继续? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 1
fi

# 更新 VERSION 文件
echo "$NEW_VERSION" > VERSION
echo "✓ 已更新 VERSION 文件"

# 检查是否在 git 仓库中
if git rev-parse --git-dir > /dev/null 2>&1; then
    echo ""
    echo "=== Git 操作 ==="
    
    # 显示当前状态
    echo "当前 Git 状态:"
    git status --short
    echo ""
    
    read -p "是否创建 Git 提交和标签? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # 提交信息
        read -p "请输入提交信息 (留空使用默认): " COMMIT_MSG
        if [ -z "$COMMIT_MSG" ]; then
            COMMIT_MSG="chore: bump version to $NEW_VERSION"
        fi
        
        # Git 操作
        git add VERSION
        git commit -m "$COMMIT_MSG"
        git tag -a "v$NEW_VERSION" -m "Release version $NEW_VERSION"
        
        echo "✓ 已创建提交和标签 v$NEW_VERSION"
        echo ""
        
        read -p "是否推送到远程仓库? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git push origin main || git push origin master
            git push origin "v$NEW_VERSION"
            echo "✓ 已推送到远程仓库"
        fi
    fi
fi

echo ""
echo "=== 构建发布包 ==="
read -p "是否构建发布包? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    make release
    echo ""
    echo "✓ 发布包已创建"
fi

echo ""
echo "=== 完成 ==="
echo "版本 $NEW_VERSION 准备就绪！"
echo ""
echo "下一步操作："
echo "1. 更新 CHANGELOG.md 记录变更"
echo "2. 测试发布包: tar -xzf release/airecorder-$NEW_VERSION.tar.gz"
echo "3. 构建 Docker 镜像: make build"
echo "4. 发布到生产环境"
echo ""
