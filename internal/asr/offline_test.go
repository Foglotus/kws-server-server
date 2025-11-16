package asr

import (
	"os"
	"path/filepath"
	"testing"

	"airecorder/internal/config"
)

func getTestAudioPath(filename string) string {
	// 尝试从环境变量获取项目根目录
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return filepath.Join(root, filename)
	}
	// 默认相对于当前工作目录
	return filepath.Join("../..", filename)
}

func TestOfflineASR(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建离线识别管理器
	manager := NewOfflineASRManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer manager.Close()

	// 测试用例
	testCases := []struct {
		name       string
		audioFile  string
		sampleRate int
	}{
		{
			name:       "Test WAV file",
			audioFile:  getTestAudioPath("test.wav"),
			sampleRate: 16000,
		},
		{
			name:       "Test MP4 file",
			audioFile:  getTestAudioPath("test.mp4"),
			sampleRate: 16000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 加载音频文件
			samples, sampleRate, err := LoadAudioFile(tc.audioFile, tc.sampleRate)
			if err != nil {
				t.Skipf("Skipping test (file may not exist or ffmpeg not available): %v", err)
				return
			}

			t.Logf("Loaded audio: %d samples at %d Hz (%.2f seconds)",
				len(samples), sampleRate, float64(len(samples))/float64(sampleRate))

			// 执行识别
			text, err := manager.Recognize(samples, sampleRate)
			if err != nil {
				t.Errorf("Recognition failed: %v", err)
				return
			}

			// 验证结果
			if text == "" {
				t.Error("Recognition returned empty text")
			} else {
				t.Logf("Recognition result: %s", text)
			}
		})
	}
}

func TestOfflineASRWithShortAudio(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建离线识别管理器
	manager := NewOfflineASRManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer manager.Close()

	// 测试空音频
	t.Run("Empty audio", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Empty audio caused panic (expected): %v", r)
			}
		}()

		samples := []float32{}
		text, err := manager.Recognize(samples, 16000)
		if err != nil {
			t.Logf("Expected behavior for empty audio: %v", err)
		} else {
			t.Logf("Empty audio result: '%s'", text)
		}
	})

	// 测试极短音频 (0.1秒)
	t.Run("Very short audio", func(t *testing.T) {
		sampleRate := 16000
		duration := 0.1
		numSamples := int(float64(sampleRate) * duration)
		samples := make([]float32, numSamples)

		// 生成简单的正弦波
		for i := range samples {
			samples[i] = 0.1 // 静音
		}

		text, err := manager.Recognize(samples, sampleRate)
		if err != nil {
			t.Logf("Short audio recognition error: %v", err)
		} else {
			t.Logf("Short audio result: '%s'", text)
		}
	})
}

func TestOfflineASRStats(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建离线识别管理器
	manager := NewOfflineASRManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer manager.Close()

	// 执行一些识别操作
	samples := make([]float32, 16000) // 1秒音频
	_, _ = manager.Recognize(samples, 16000)

	// 获取统计信息
	stats := manager.GetStats()
	if stats == nil {
		t.Error("GetStats returned nil")
		return
	}

	t.Logf("Stats: %+v", stats)

	// 验证统计信息
	if totalReqs, ok := stats["total_requests"].(int64); ok {
		if totalReqs < 1 {
			t.Error("Expected at least 1 total request")
		}
	} else {
		t.Error("Missing or invalid total_requests in stats")
	}
}

func BenchmarkOfflineASR(b *testing.B) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	// 创建离线识别管理器
	manager := NewOfflineASRManager(cfg)
	if manager == nil {
		b.Fatal("Failed to create offline ASR manager")
	}
	defer manager.Close()

	// 尝试加载测试音频
	samples, sampleRate, err := LoadAudioFile(getTestAudioPath("test.wav"), 16000)
	if err != nil {
		// 如果没有测试文件，使用生成的音频
		sampleRate = 16000
		samples = make([]float32, sampleRate*3) // 3秒音频
		b.Logf("Using generated audio for benchmark")
	} else {
		b.Logf("Using test.wav for benchmark: %d samples", len(samples))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.Recognize(samples, sampleRate)
		if err != nil {
			b.Errorf("Recognition failed: %v", err)
		}
	}
}
