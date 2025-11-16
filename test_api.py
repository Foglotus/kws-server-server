#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
AI Recorder 测试脚本
测试离线语音识别和说话者分离功能
"""

import requests
import base64
import json
import sys
import argparse
from pathlib import Path


def read_audio_file(file_path):
    """读取音频文件并返回 Base64 编码"""
    with open(file_path, 'rb') as f:
        audio_data = f.read()
    return base64.b64encode(audio_data).decode('utf-8')


def test_offline_asr(server_url, audio_file, sample_rate=16000):
    """测试离线语音识别"""
    print(f"测试离线语音识别: {audio_file}")
    print("-" * 50)
    
    # 读取音频
    audio_base64 = read_audio_file(audio_file)
    
    # 发送请求
    url = f"{server_url}/api/v1/offline/asr"
    payload = {
        "audio": audio_base64,
        "sample_rate": sample_rate
    }
    
    try:
        response = requests.post(url, json=payload, timeout=30)
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ 识别成功")
        print(f"  文本: {result.get('text', '')}")
        print(f"  时长: {result.get('duration', 0):.2f} 秒")
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求失败: {e}")
        return False


def test_diarization_asr(server_url, audio_file, sample_rate=16000):
    """测试带说话者分离的语音识别"""
    print(f"\n测试说话者分离识别: {audio_file}")
    print("-" * 50)
    
    # 读取音频
    audio_base64 = read_audio_file(audio_file)
    
    # 发送请求
    url = f"{server_url}/api/v1/offline/asr/diarization"
    payload = {
        "audio": audio_base64,
        "sample_rate": sample_rate
    }
    
    try:
        response = requests.post(url, json=payload, timeout=60)
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ 识别成功")
        print(f"  完整文本: {result.get('text', '')}")
        print(f"  时长: {result.get('duration', 0):.2f} 秒")
        
        segments = result.get('segments', [])
        if segments:
            print(f"\n  说话者片段 (共 {len(segments)} 个):")
            for seg in segments:
                print(f"    [{seg['start']:.2f}s - {seg['end']:.2f}s] "
                      f"说话者 {seg['speaker']}: {seg['text']}")
        
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求失败: {e}")
        return False


def test_diarization_only(server_url, audio_file, sample_rate=16000):
    """测试纯说话者分离"""
    print(f"\n测试纯说话者分离: {audio_file}")
    print("-" * 50)
    
    # 读取音频
    audio_base64 = read_audio_file(audio_file)
    
    # 发送请求
    url = f"{server_url}/api/v1/diarization"
    payload = {
        "audio": audio_base64,
        "sample_rate": sample_rate
    }
    
    try:
        response = requests.post(url, json=payload, timeout=60)
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ 分离成功")
        print(f"  时长: {result.get('duration', 0):.2f} 秒")
        
        segments = result.get('segments', [])
        if segments:
            print(f"\n  说话者片段 (共 {len(segments)} 个):")
            for seg in segments:
                print(f"    [{seg['start']:.2f}s - {seg['end']:.2f}s] "
                      f"说话者 {seg['speaker']}")
        
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"✗ 请求失败: {e}")
        return False


def test_health_check(server_url):
    """测试健康检查"""
    print("测试健康检查")
    print("-" * 50)
    
    try:
        response = requests.get(f"{server_url}/health", timeout=5)
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ 服务健康")
        print(f"  状态: {result.get('status', '')}")
        print(f"  服务: {result.get('service', '')}")
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"✗ 健康检查失败: {e}")
        return False


def test_stats(server_url):
    """测试统计信息"""
    print("\n测试统计信息")
    print("-" * 50)
    
    try:
        response = requests.get(f"{server_url}/api/v1/stats", timeout=5)
        response.raise_for_status()
        
        result = response.json()
        print(f"✓ 统计信息获取成功")
        print(json.dumps(result, indent=2, ensure_ascii=False))
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"✗ 统计信息获取失败: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description='AI Recorder 测试工具')
    parser.add_argument('--server', default='http://localhost:11123',
                        help='服务器地址 (默认: http://localhost:11123)')
    parser.add_argument('--audio', type=str,
                        help='测试音频文件路径')
    parser.add_argument('--sample-rate', type=int, default=16000,
                        help='音频采样率 (默认: 16000)')
    parser.add_argument('--test', choices=['all', 'health', 'asr', 'diarization', 'stats'],
                        default='all',
                        help='测试类型')
    
    args = parser.parse_args()
    
    print("=" * 50)
    print("AI Recorder 测试工具")
    print("=" * 50)
    print(f"服务器: {args.server}\n")
    
    # 健康检查
    if args.test in ['all', 'health']:
        if not test_health_check(args.server):
            print("\n✗ 服务未运行或不健康，请检查服务状态")
            sys.exit(1)
    
    # 如果提供了音频文件，进行识别测试
    if args.audio and args.test in ['all', 'asr', 'diarization']:
        audio_file = Path(args.audio)
        
        if not audio_file.exists():
            print(f"\n✗ 音频文件不存在: {audio_file}")
            sys.exit(1)
        
        # 离线识别测试
        if args.test in ['all', 'asr']:
            test_offline_asr(args.server, audio_file, args.sample_rate)
        
        # 说话者分离测试
        if args.test in ['all', 'diarization']:
            test_diarization_only(args.server, audio_file, args.sample_rate)
            test_diarization_asr(args.server, audio_file, args.sample_rate)
    
    # 统计信息测试
    if args.test in ['all', 'stats']:
        test_stats(args.server)
    
    print("\n" + "=" * 50)
    print("测试完成")
    print("=" * 50)


if __name__ == '__main__':
    main()
