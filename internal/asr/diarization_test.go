package asr

import (
	"testing"

	"airecorder/internal/config"
)

func TestDiarization(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建说话者分离管理器
	diarizationMgr := NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		t.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 测试用例
	testCases := []struct {
		name       string
		audioFile  string
		sampleRate int
	}{
		{
			name:       "Test WAV file diarization",
			audioFile:  getTestAudioPath("test.wav"),
			sampleRate: 16000,
		},
		{
			name:       "Test MP4 file diarization",
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

			// 执行说话者分离
			segments, err := diarizationMgr.Process(samples, sampleRate)
			if err != nil {
				t.Errorf("Diarization failed: %v", err)
				return
			}

			// 验证结果
			if len(segments) == 0 {
				t.Log("No speaker segments detected (audio may be too short or silent)")
			} else {
				t.Logf("Found %d speaker segments:", len(segments))
				for i, seg := range segments {
					t.Logf("  Segment %d: Speaker %d, Time: %.2fs - %.2fs (%.2fs)",
						i, seg.Speaker, seg.Start, seg.End, seg.End-seg.Start)
				}

				// 统计每个说话者的总时长
				speakerDuration := make(map[int]float32)
				for _, seg := range segments {
					speakerDuration[seg.Speaker] += seg.End - seg.Start
				}

				t.Logf("Speaker statistics:")
				for speaker, duration := range speakerDuration {
					t.Logf("  Speaker %d: %.2fs total", speaker, duration)
				}
			}
		})
	}
}

func TestDiarizationWithASR(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建说话者分离管理器
	diarizationMgr := NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		t.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 创建离线识别管理器
	asrMgr := NewOfflineASRManager(cfg)
	if asrMgr == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	// 测试用例
	testCases := []struct {
		name       string
		audioFile  string
		sampleRate int
	}{
		{
			name:       "Test WAV with ASR",
			audioFile:  getTestAudioPath("test.wav"),
			sampleRate: 16000,
		},
		{
			name:       "Test MP4 with ASR",
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

			// 执行说话者分离 + ASR
			segments, err := diarizationMgr.ProcessWithASR(samples, sampleRate, asrMgr)
			if err != nil {
				t.Errorf("Diarization with ASR failed: %v", err)
				return
			}

			// 验证结果
			if len(segments) == 0 {
				t.Log("No speaker segments detected")
			} else {
				t.Logf("Found %d speaker segments with transcription:", len(segments))
				for i, seg := range segments {
					t.Logf("  Segment %d:", i)
					t.Logf("    Speaker: %d", seg.Speaker)
					t.Logf("    Time: %.2fs - %.2fs (%.2fs)", seg.Start, seg.End, seg.End-seg.Start)
					t.Logf("    Text: %s", seg.Text)
				}

				// 生成完整的转录文本
				fullText := ""
				for _, seg := range segments {
					fullText += seg.Text + " "
				}
				t.Logf("\nFull transcription: %s", fullText)
			}
		})
	}
}

func TestDiarizationSampleRate(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建说话者分离管理器
	diarizationMgr := NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		t.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 获取期望的采样率
	expectedSampleRate := diarizationMgr.diarization.SampleRate()
	t.Logf("Diarization expected sample rate: %d Hz", expectedSampleRate)

	// 测试错误的采样率
	t.Run("Wrong sample rate", func(t *testing.T) {
		samples := make([]float32, 16000)
		wrongSampleRate := expectedSampleRate + 1000

		_, err := diarizationMgr.Process(samples, wrongSampleRate)
		if err == nil {
			t.Error("Expected error for wrong sample rate, but got nil")
		} else {
			t.Logf("Correctly rejected wrong sample rate: %v", err)
		}
	})

	// 测试正确的采样率
	t.Run("Correct sample rate", func(t *testing.T) {
		samples := make([]float32, expectedSampleRate*2) // 2秒音频

		segments, err := diarizationMgr.Process(samples, expectedSampleRate)
		if err != nil {
			t.Errorf("Failed with correct sample rate: %v", err)
		} else {
			t.Logf("Successfully processed with correct sample rate, found %d segments", len(segments))
		}
	})
}

func BenchmarkDiarization(b *testing.B) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	// 创建说话者分离管理器
	diarizationMgr := NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		b.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 尝试加载测试音频
	samples, sampleRate, err := LoadAudioFile(getTestAudioPath("test.wav"), 16000)
	if err != nil {
		// 如果没有测试文件，使用生成的音频
		sampleRate = 16000
		samples = make([]float32, sampleRate*5) // 5秒音频
		b.Logf("Using generated audio for benchmark")
	} else {
		b.Logf("Using test.wav for benchmark: %d samples", len(samples))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := diarizationMgr.Process(samples, sampleRate)
		if err != nil {
			b.Errorf("Diarization failed: %v", err)
		}
	}
}

func BenchmarkDiarizationWithASR(b *testing.B) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	// 创建管理器
	diarizationMgr := NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		b.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	asrMgr := NewOfflineASRManager(cfg)
	if asrMgr == nil {
		b.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	// 尝试加载测试音频
	samples, sampleRate, err := LoadAudioFile(getTestAudioPath("test.wav"), 16000)
	if err != nil {
		// 如果没有测试文件，使用生成的音频
		sampleRate = 16000
		samples = make([]float32, sampleRate*5) // 5秒音频
		b.Logf("Using generated audio for benchmark")
	} else {
		b.Logf("Using test.wav for benchmark: %d samples", len(samples))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := diarizationMgr.ProcessWithASR(samples, sampleRate, asrMgr)
		if err != nil {
			b.Errorf("Diarization with ASR failed: %v", err)
		}
	}
}
