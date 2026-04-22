#!/bin/bash

# 启动服务并打开测试页面

echo "🚀 正在启动 AI Recorder 服务..."

# 检查配置文件
if [ ! -f "config.yaml" ]; then
    echo "❌ 错误: 找不到 config.yaml 配置文件"
    exit 1
fi

# 检查模型文件
if [ ! -d "models" ]; then
    echo "❌ 错误: 找不到 models 目录"
    echo "请先运行: ./download_models.sh"
    exit 1
fi

# 启动服务
echo "📦 启动 Docker 容器..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 3

# 检查服务状态
MAX_RETRIES=10
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:11123/realkws/health > /dev/null 2>&1; then
        echo "✅ 服务启动成功！"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "等待服务响应... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ 服务启动失败，请查看日志："
    echo "   docker-compose logs"
    exit 1
fi

# 显示服务信息
echo ""
echo "================================================"
echo "  🎉 AI Recorder 服务已启动！"
echo "================================================"
echo ""
echo "📍 服务地址:"
echo "   - Web 测试页面: http://localhost:11123/test"
echo "   - API 基础地址: http://localhost:11123"
echo "   - 健康检查: http://localhost:11123/realkws/health"
echo ""
echo "📚 文档:"
echo "   - API 文档: API_DOCS.md"
echo "   - 测试指南: TESTING.md"
echo "   - 开发文档: DEVELOPMENT.md"
echo ""
echo "🛠️  常用命令:"
echo "   - 查看日志: docker-compose logs -f"
echo "   - 停止服务: docker-compose down"
echo "   - 重启服务: docker-compose restart"
echo ""
echo "================================================"
echo ""

# 尝试在浏览器中打开测试页面
if command -v open > /dev/null 2>&1; then
    # macOS
    echo "🌐 正在打开测试页面..."
    sleep 2
    open "http://localhost:11123/test"
elif command -v xdg-open > /dev/null 2>&1; then
    # Linux
    echo "🌐 正在打开测试页面..."
    sleep 2
    xdg-open "http://localhost:11123/test"
else
    echo "💡 请在浏览器中打开: http://localhost:11123/test"
fi
