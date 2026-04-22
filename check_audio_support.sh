#!/bin/bash
# 音频格式支持快速测试脚本

echo "================================"
echo "音频格式支持测试"
echo "================================"
echo ""

# 检查 FFmpeg 是否安装
echo "1. 检查 FFmpeg 安装状态..."
if command -v ffmpeg &> /dev/null; then
    echo "✓ FFmpeg 已安装"
    ffmpeg -version | head -n 1
    echo ""
    echo "支持的音频格式："
    echo "  - WAV (原生支持)"
    echo "  - MP3 (通过 FFmpeg)"
    echo "  - M4A/MP4 (通过 FFmpeg)"
    echo "  - FLAC (通过 FFmpeg)"
    echo "  - OGG/OPUS (通过 FFmpeg)"
    echo "  - AAC (通过 FFmpeg)"
    echo "  - WMA (通过 FFmpeg)"
    echo "  - AMR (通过 FFmpeg)"
else
    echo "✗ FFmpeg 未安装"
    echo ""
    echo "当前只支持 WAV 格式"
    echo ""
    echo "安装 FFmpeg 以支持更多格式："
    echo "  macOS:   brew install ffmpeg"
    echo "  Ubuntu:  sudo apt-get install ffmpeg"
    echo "  CentOS:  sudo yum install ffmpeg"
fi

echo ""
echo "================================"
echo "2. 测试示例"
echo "================================"
echo ""

# 创建测试音频文件（如果 FFmpeg 可用）
if command -v ffmpeg &> /dev/null; then
    echo "生成测试音频文件..."
    
    # 生成一个简单的测试音频（静音）
    ffmpeg -f lavfi -i "anullsrc=r=16000:cl=mono" -t 2 -ar 16000 -ac 1 test_wav.wav -y &> /dev/null
    
    if [ -f "test_wav.wav" ]; then
        echo "✓ 已生成 test_wav.wav (2秒静音)"
        
        # 转换为其他格式
        ffmpeg -i test_wav.wav -ar 16000 -ac 1 test_mp3.mp3 -y &> /dev/null
        echo "✓ 已生成 test_mp3.mp3"
        
        ffmpeg -i test_wav.wav -ar 16000 -ac 1 test_m4a.m4a -y &> /dev/null
        echo "✓ 已生成 test_m4a.m4a"
        
        ffmpeg -i test_wav.wav -ar 16000 -ac 1 test_flac.flac -y &> /dev/null
        echo "✓ 已生成 test_flac.flac"
        
        echo ""
        echo "测试文件已生成，现在可以测试上传："
        echo ""
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_wav.wav'"
        echo ""
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_mp3.mp3'"
        echo ""
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_m4a.m4a'"
        echo ""
        echo "  curl -X POST http://localhost:11123/api/v1/offline/asr \\"
        echo "    -F 'audio_file=@test_flac.flac'"
        echo ""
        echo "或使用 Python 测试脚本："
        echo "  python test_audio_formats.py test_wav.wav test_mp3.mp3 test_m4a.m4a"
    fi
fi

echo ""
echo "================================"
echo "3. 查看服务支持的格式"
echo "================================"
echo ""
echo "启动服务后，访问："
echo "  curl http://localhost:11123/"
echo ""
echo "响应中的 'supported_audio_formats' 字段会列出所有支持的格式"
echo ""

echo "================================"
echo "更多信息"
echo "================================"
echo ""
echo "详细文档："
echo "  - API 文档: docs/API_DOCS.md"
echo "  - 音频格式说明: docs/AUDIO_FORMATS.md"
echo "  - Python 测试: test_audio_formats.py"
echo ""
