#!/bin/bash

# 快速更新脚本 - 仅更新程序文件
# 适用于代码修改但模型和配置未变的场景

set -e

echo "=== AI Recorder 快速更新 ==="
echo ""

# 检查参数
REMOTE_HOST=""
REMOTE_PATH=""

if [ $# -eq 2 ]; then
    REMOTE_HOST=$1
    REMOTE_PATH=$2
else
    echo "用法 1 (远程更新): $0 <远程主机> <远程路径>"
    echo "示例: $0 user@192.168.1.100 /opt/airecorder"
    echo ""
    echo "用法 2 (本地更新): $0"
    echo ""
    read -p "选择更新方式 [1:远程, 2:本地]: " choice
    
    if [ "$choice" == "1" ]; then
        read -p "远程主机 (如 user@192.168.1.100): " REMOTE_HOST
        read -p "远程路径 (如 /opt/airecorder): " REMOTE_PATH
    fi
fi

# 步骤 1: 编译新的二进制文件
echo "步骤 1: 编译新的二进制文件..."
if [ ! -f "./build_binary.sh" ]; then
    echo "错误: build_binary.sh 不存在"
    exit 1
fi

chmod +x ./build_binary.sh
./build_binary.sh

if [ ! -f "./bin/airecorder" ]; then
    echo "错误: 编译失败"
    exit 1
fi

echo ""
echo "✓ 编译完成"
echo ""

# 步骤 2: 打包更新文件
echo "步骤 2: 打包更新文件..."
mkdir -p ./update_package
cp ./bin/airecorder ./update_package/
cp -r ./bin/lib ./update_package/ 2>/dev/null || true
cp ./config.yaml ./update_package/ 2>/dev/null || true
cp -r ./static ./update_package/ 2>/dev/null || true

# 创建更新脚本
cat > ./update_package/apply_update.sh << 'EOF'
#!/bin/bash
set -e
echo "=== 应用更新 ==="
echo "停止服务..."
docker-compose -f docker-compose.runtime.yml down || docker-compose down

echo "备份当前版本..."
if [ -f "./bin/airecorder" ]; then
    cp ./bin/airecorder ./bin/airecorder.backup.$(date +%Y%m%d_%H%M%S)
fi

echo "复制新文件..."
cp -f airecorder ../bin/
cp -rf lib/* ../bin/lib/ 2>/dev/null || true
cp -f config.yaml ../ 2>/dev/null || true
cp -rf static/* ../static/ 2>/dev/null || true

echo "设置权限..."
chmod +x ../bin/airecorder

echo "启动服务..."
cd ..
docker-compose -f docker-compose.runtime.yml up -d

echo ""
echo "✓ 更新完成！"
echo "检查服务状态..."
sleep 5
docker-compose -f docker-compose.runtime.yml ps
EOF

chmod +x ./update_package/apply_update.sh

# 压缩更新包
cd update_package
tar -czf ../airecorder_update_$(cat ../VERSION).tar.gz .
cd ..

echo ""
echo "✓ 更新包已创建: airecorder_update_$(cat VERSION).tar.gz"
echo ""

# 步骤 3: 传输和应用更新
if [ -n "$REMOTE_HOST" ] && [ -n "$REMOTE_PATH" ]; then
    echo "步骤 3: 传输到远程服务器..."
    
    UPDATE_FILE="airecorder_update_$(cat VERSION).tar.gz"
    
    echo "上传文件到 ${REMOTE_HOST}:${REMOTE_PATH}..."
    scp "$UPDATE_FILE" "${REMOTE_HOST}:${REMOTE_PATH}/"
    
    echo ""
    read -p "是否立即在远程服务器应用更新？ (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "远程执行更新..."
        ssh "$REMOTE_HOST" << ENDSSH
cd ${REMOTE_PATH}
mkdir -p update_tmp
cd update_tmp
tar -xzf ../${UPDATE_FILE}
bash apply_update.sh
cd ..
rm -rf update_tmp
ENDSSH
        echo ""
        echo "✓ 远程更新完成！"
    else
        echo ""
        echo "更新包已上传，手动执行以下命令应用更新："
        echo "  cd ${REMOTE_PATH}"
        echo "  mkdir -p update_tmp && cd update_tmp"
        echo "  tar -xzf ../${UPDATE_FILE}"
        echo "  bash apply_update.sh"
    fi
else
    echo "步骤 3: 本地更新..."
    read -p "是否立即应用更新？ (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd update_package
        bash apply_update.sh
        cd ..
        echo ""
        echo "✓ 本地更新完成！"
    else
        echo ""
        echo "更新包位置: ./update_package/"
        echo "手动应用: cd update_package && bash apply_update.sh"
    fi
fi

# 清理
rm -rf update_package

echo ""
echo "=== 更新流程完成 ==="
echo ""
echo "文件大小对比："
echo "  完整部署包: ~2GB (包含镜像+模型)"
echo "  快速更新包: ~$(du -sh airecorder_update_*.tar.gz 2>/dev/null | cut -f1) (仅程序文件)"
echo ""
