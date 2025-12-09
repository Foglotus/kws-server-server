#!/usr/bin/env python3
"""
音频格式支持测试脚本
测试不同音频格式的识别能力
"""

import requests
import base64
import json
import sys
import os
from pathlib import Path

# 配置
API_BASE_URL = "http://localhost:11123"
API_ENDPOINT = f"{API_BASE_URL}/api/v1/offline/asr"

def check_service():
    """检查服务是否运行"""
    try:
        response = requests.get(API_BASE_URL)
        if response.status_code == 200:
            data = response.json()
            print("✓ 服务运行正常")
            print(f"  版本: {data.get('version', 'unknown')}")
            print(f"  支持的音频格式: {', '.join(data.get('supported_audio_formats', []))}")
            return True
    except requests.exceptions.ConnectionError:
        print("✗ 无法连接到服务")
        print(f"  请确保服务运行在 {API_BASE_URL}")
        return False
    return False

def test_file_upload(audio_file):
    """测试文件上传方式"""
    if not os.path.exists(audio_file):
        print(f"✗ 文件不存在: {audio_file}")
        return False
    
    file_name = os.path.basename(audio_file)
    file_ext = os.path.splitext(audio_file)[1].lower()
    
    print(f"\n{'='*60}")
    print(f"测试文件: {file_name}")
    print(f"格式: {file_ext}")
    print(f"大小: {os.path.getsize(audio_file)} 字节")
    print(f"{'-'*60}")
    
    try:
        # 方式1: 文件上传
        with open(audio_file, 'rb') as f:
            files = {'audio_file': (file_name, f, 'audio/*')}
            response = requests.post(API_ENDPOINT, files=files, timeout=60)
        
        if response.status_code == 200:
            result = response.json()
            print("✓ 识别成功 (文件上传方式)")
            print(f"  识别结果: {result.get('text', '')}")
            print(f"  音频时长: {result.get('duration', 0):.2f} 秒")
            return True
        else:
            print(f"✗ 识别失败 (HTTP {response.status_code})")
            print(f"  错误信息: {response.json().get('error', '未知错误')}")
            return False
            
    except requests.exceptions.Timeout:
        print("✗ 请求超时")
        return False
    except Exception as e:
        print(f"✗ 发生错误: {str(e)}")
        return False

def test_base64_upload(audio_file):
    """测试 Base64 编码方式"""
    if not os.path.exists(audio_file):
        return False
    
    file_name = os.path.basename(audio_file)
    
    try:
        # 方式2: Base64 编码
        with open(audio_file, 'rb') as f:
            audio_data = f.read()
        
        audio_base64 = base64.b64encode(audio_data).decode('utf-8')
        
        payload = {
            "audio": audio_base64,
            "sample_rate": 16000
        }
        
        headers = {'Content-Type': 'application/json'}
        response = requests.post(API_ENDPOINT, 
                                json=payload, 
                                headers=headers,
                                timeout=60)
        
        if response.status_code == 200:
            result = response.json()
            print("✓ 识别成功 (Base64 编码方式)")
            print(f"  识别结果: {result.get('text', '')}")
            print(f"  音频时长: {result.get('duration', 0):.2f} 秒")
            return True
        else:
            print(f"✗ 识别失败 (HTTP {response.status_code})")
            print(f"  错误信息: {response.json().get('error', '未知错误')}")
            return False
            
    except Exception as e:
        print(f"✗ Base64 方式错误: {str(e)}")
        return False

def test_diarization(audio_file):
    """测试说话者分离功能"""
    if not os.path.exists(audio_file):
        return False
    
    file_name = os.path.basename(audio_file)
    endpoint = f"{API_BASE_URL}/api/v1/offline/asr/diarization"
    
    print(f"\n{'='*60}")
    print(f"测试说话者分离: {file_name}")
    print(f"{'-'*60}")
    
    try:
        with open(audio_file, 'rb') as f:
            files = {'audio_file': (file_name, f, 'audio/*')}
            response = requests.post(endpoint, files=files, timeout=120)
        
        if response.status_code == 200:
            result = response.json()
            print("✓ 说话者分离成功")
            print(f"  完整文本: {result.get('text', '')}")
            print(f"  音频时长: {result.get('duration', 0):.2f} 秒")
            
            segments = result.get('segments', [])
            if segments:
                print(f"  说话者片段 ({len(segments)} 个):")
                for i, seg in enumerate(segments[:5]):  # 只显示前5个
                    print(f"    {i+1}. [{seg['start']:.2f}s - {seg['end']:.2f}s] "
                          f"说话者{seg['speaker']}: {seg['text']}")
                if len(segments) > 5:
                    print(f"    ... 还有 {len(segments)-5} 个片段")
            return True
        else:
            print(f"✗ 说话者分离失败 (HTTP {response.status_code})")
            print(f"  错误信息: {response.json().get('error', '未知错误')}")
            return False
            
    except Exception as e:
        print(f"✗ 发生错误: {str(e)}")
        return False

def main():
    print("="*60)
    print("音频格式支持测试")
    print("="*60)
    
    # 检查服务
    if not check_service():
        sys.exit(1)
    
    # 测试文件
    if len(sys.argv) < 2:
        print("\n使用方法:")
        print(f"  python {sys.argv[0]} <音频文件1> [音频文件2] ...")
        print("\n支持的格式:")
        print("  WAV, MP3, M4A, FLAC, OGG, OPUS, AAC, WMA, AMR")
        print("\n示例:")
        print(f"  python {sys.argv[0]} test.wav")
        print(f"  python {sys.argv[0]} recording.mp3 voice.m4a audio.flac")
        sys.exit(0)
    
    audio_files = sys.argv[1:]
    
    # 统计结果
    total = len(audio_files)
    success = 0
    
    for audio_file in audio_files:
        if test_file_upload(audio_file):
            success += 1
            # 可选：测试 Base64 方式
            # test_base64_upload(audio_file)
    
    # 汇总
    print(f"\n{'='*60}")
    print("测试汇总")
    print(f"{'-'*60}")
    print(f"总计: {total} 个文件")
    print(f"成功: {success} 个")
    print(f"失败: {total - success} 个")
    print(f"成功率: {success/total*100:.1f}%")
    print(f"{'='*60}")
    
    # 可选：测试说话者分离（如果有文件）
    if audio_files and success > 0:
        print("\n是否测试说话者分离功能？(y/n): ", end='')
        try:
            choice = input().strip().lower()
            if choice == 'y':
                test_diarization(audio_files[0])
        except:
            pass

if __name__ == "__main__":
    main()
