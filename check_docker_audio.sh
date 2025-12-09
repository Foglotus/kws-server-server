#!/bin/bash
# Docker 音频格式支持验证脚本

echo "================================"
echo "Docker 环境音频格式支持检查"
echo "================================"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "✗ Docker 未运行或未安装"
    echo "  请先启动 Docker"
    exit 1
fi

echo "✓ Docker 正在运行"
echo ""

# 检查是否有正在运行的容器
CONTAINER_NAME="airecorder"
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "✓ 找到运行中的容器: $CONTAINER_NAME"
    echo ""
    
    # 检查容器内的 FFmpeg
    echo "1. 检查容器内 FFmpeg 安装状态..."
    if docker exec $CONTAINER_NAME which ffmpeg > /dev/null 2>&1; then
        echo "✓ FFmpeg 已安装"
        docker exec $CONTAINER_NAME ffmpeg -version | head -n 1
        echo ""
        
        echo "2. 支持的音频格式："
        echo "  ✓ WAV (原生支持)"
        echo "  ✓ MP3 (通过 FFmpeg)"
        echo "  ✓ M4A/MP4 (通过 FFmpeg)"
        echo "  ✓ FLAC (通过 FFmpeg)"
        echo "  ✓ OGG/OPUS (通过 FFmpeg)"
        echo "  ✓ AAC (通过 FFmpeg)"
        echo "  ✓ WMA (通过 FFmpeg)"
        echo "  ✓ AMR (通过 FFmpeg)"
    else
        echo "✗ FFmpeg 未安装"
        echo "  容器内只支持 WAV 格式"
        echo ""
        echo "建议重新构建镜像："
        echo "  docker-compose build --no-cache"
    fi
    
    echo ""
    echo "3. 查询服务支持的格式..."
    RESPONSE=$(curl -s http://localhost:11123/)
    if [ $? -eq 0 ]; then
        echo "✓ 服务可访问"
        echo ""
        echo "服务响应："
        echo "$RESPONSE" | grep -o '"supported_audio_formats":\[[^]]*\]' || echo "  (未找到格式信息，可能服务版本较旧)"
    else
        echo "✗ 无法访问服务"
        echo "  请确保服务正在运行且端口 11123 可访问"
    fi
    
else
    echo "✗ 未找到运行中的容器: $CONTAINER_NAME"
    echo ""
    echo "请先启动容器："
    echo "  docker-compose up -d"
    exit 1
fi

echo ""
echo "================================"
echo "4. 测试音频格式转换"
echo "================================"
echo ""

# 生成测试音频文件
if command -v ffmpeg &> /dev/null; then
    echo "在本地生成测试音频..."
    
    # 生成测试 WAV
    ffmpeg -f lavfi -i "anullsrc=r=16000:cl=mono" -t 1 -ar 16000 -ac 1 test_docker.wav -y &> /dev/null
    
    if [ -f "test_docker.wav" ]; then
        echo "✓ 生成测试文件: test_docker.wav"
        
        # 转换为其他格式
        ffmpeg -i test_docker.wav test_docker.mp3 -y &> /dev/null
        echo "✓ 生成测试文件: test_docker.mp3"
        
        echo ""
        echo "现在可以测试上传到容器中的服务："
        echo ""
        echo "  # 测试 WAV"
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_docker.wav'"
        echo ""
        echo "  # 测试 MP3"
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_docker.mp3'"
        echo ""
        
        # 可选：自动测试
        read -p "是否立即测试上传？(y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo ""
            echo "测试 WAV 上传..."
            RESULT=$(curl -s -X POST http://localhost:11123/api/v1/offline/asr \
                -F "audio_file=@test_docker.wav")
            if echo "$RESULT" | grep -q "text"; then
                echo "✓ WAV 格式测试成功"
                echo "  响应: $RESULT"
            else
                echo "✗ WAV 格式测试失败"
                echo "  响应: $RESULT"
            fi
            
            echo ""
            echo "测试 MP3 上传..."
            RESULT=$(curl -s -X POST http://localhost:11123/api/v1/offline/asr \
                -F "audio_file=@test_docker.mp3")
            if echo "$RESULT" | grep -q "text"; then
                echo "✓ MP3 格式测试成功"
                echo "  响应: $RESULT"
            else
                echo "✗ MP3 格式测试失败"
                echo "  响应: $RESULT"
            fi
        fi
    fi
else
    echo "本地未安装 FFmpeg，跳过测试文件生成"
fi

echo ""
echo "================================"
echo "检查完成"
echo "================================"
echo ""
echo "相关命令："
echo "  - 查看容器日志: docker logs $CONTAINER_NAME"
echo "  - 进入容器: docker exec -it $CONTAINER_NAME /bin/bash"
echo "  - 重启容器: docker-compose restart"
echo "  - 重新构建: docker-compose build --no-cache"
echo ""
