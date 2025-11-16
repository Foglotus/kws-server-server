#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
WebSocket 实时语音识别测试
"""

import asyncio
import websockets
import json
import base64
import wave
import sys
from pathlib import Path


async def stream_audio_file(websocket, audio_file, chunk_duration=0.1):
    """
    将音频文件分块发送到 WebSocket
    
    Args:
        websocket: WebSocket 连接
        audio_file: 音频文件路径
        chunk_duration: 每个块的时长（秒）
    """
    # 打开音频文件
    with wave.open(str(audio_file), 'rb') as wf:
        sample_rate = wf.getframerate()
        sample_width = wf.getsampwidth()
        channels = wf.getnchannels()
        
        print(f"音频信息:")
        print(f"  采样率: {sample_rate} Hz")
        print(f"  采样宽度: {sample_width} bytes")
        print(f"  声道数: {channels}")
        print()
        
        # 计算每个块的帧数
        chunk_frames = int(sample_rate * chunk_duration)
        chunk_size = chunk_frames * sample_width * channels
        
        print("开始发送音频数据...")
        
        frame_count = 0
        while True:
            # 读取一块音频数据
            audio_data = wf.readframes(chunk_frames)
            if not audio_data:
                break
            
            # Base64 编码
            audio_base64 = base64.b64encode(audio_data).decode('utf-8')
            
            # 发送到服务器
            message = {
                "type": "audio",
                "audio": audio_base64,
                "sample_rate": sample_rate
            }
            
            await websocket.send(json.dumps(message))
            frame_count += 1
            
            # 模拟实时流
            await asyncio.sleep(chunk_duration)
        
        print(f"音频发送完成 (共 {frame_count} 块)")
        
        # 发送停止命令
        await websocket.send(json.dumps({
            "type": "control",
            "command": "stop"
        }))


async def receive_results(websocket):
    """接收并打印识别结果"""
    print("\n识别结果:")
    print("-" * 50)
    
    segment_idx = 0
    
    try:
        async for message in websocket:
            result = json.loads(message)
            
            if result.get('type') == 'error':
                print(f"✗ 错误: {result.get('error', '')}")
                break
            
            if result.get('type') == 'partial':
                # 实时部分结果
                text = result.get('text', '')
                if text:
                    print(f"\r[部分] {text}", end='', flush=True)
            
            elif result.get('type') == 'result':
                # 完整结果
                text = result.get('text', '')
                is_endpoint = result.get('is_endpoint', False)
                
                if is_endpoint and text:
                    segment_idx = result.get('segment', segment_idx)
                    print(f"\n[片段 {segment_idx}] {text}")
                elif text == "Session stopped":
                    print("\n\n连接已关闭")
                    break
    
    except websockets.exceptions.ConnectionClosed:
        print("\n\n连接已关闭")


async def test_streaming_asr(server_url, audio_file):
    """测试实时语音识别"""
    ws_url = server_url.replace('http://', 'ws://').replace('https://', 'wss://')
    ws_url = f"{ws_url}/api/v1/streaming/asr"
    
    print("=" * 50)
    print("WebSocket 实时语音识别测试")
    print("=" * 50)
    print(f"服务器: {ws_url}")
    print(f"音频文件: {audio_file}")
    print()
    
    try:
        async with websockets.connect(ws_url) as websocket:
            # 接收欢迎消息
            welcome = await websocket.recv()
            print(f"连接成功: {json.loads(welcome).get('text', '')}\n")
            
            # 创建任务
            send_task = asyncio.create_task(stream_audio_file(websocket, audio_file))
            receive_task = asyncio.create_task(receive_results(websocket))
            
            # 等待任务完成
            await asyncio.gather(send_task, receive_task)
            
    except Exception as e:
        print(f"\n✗ 连接失败: {e}")
        sys.exit(1)


def main():
    import argparse
    
    parser = argparse.ArgumentParser(description='WebSocket 实时识别测试')
    parser.add_argument('--server', default='http://localhost:11123',
                        help='服务器地址 (默认: http://localhost:11123)')
    parser.add_argument('--audio', type=str, required=True,
                        help='测试音频文件路径 (WAV 格式)')
    parser.add_argument('--chunk', type=float, default=0.1,
                        help='数据块时长（秒，默认: 0.1）')
    
    args = parser.parse_args()
    
    audio_file = Path(args.audio)
    if not audio_file.exists():
        print(f"✗ 音频文件不存在: {audio_file}")
        sys.exit(1)
    
    # 运行测试
    asyncio.run(test_streaming_asr(args.server, audio_file))


if __name__ == '__main__':
    main()
