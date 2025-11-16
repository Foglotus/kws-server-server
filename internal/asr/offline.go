package asr

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"

	"airecorder/internal/config"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// OfflineASRManager 离线识别管理器
type OfflineASRManager struct {
	config      *config.Config
	recognizer  *sherpa.OfflineRecognizer
	punctuation *PunctuationManager
	mu          sync.Mutex
	stats       struct {
		totalRequests int64
		totalDuration float64
		successCount  int64
		failureCount  int64
	}
}

// NewOfflineASRManager 创建离线识别管理器
func NewOfflineASRManager(cfg *config.Config) *OfflineASRManager {
	log.Println("Initializing Offline ASR Manager...")

	// 构建模型路径
	modelsDir := cfg.OfflineASR.ModelsDir

	recognizerConfig := sherpa.OfflineRecognizerConfig{}

	// 根据模型类型配置
	switch cfg.OfflineASR.ModelType {
	case "paraformer":
		// Paraformer uses a single model file
		modelPath := filepath.Join(modelsDir, cfg.OfflineASR.Encoder)
		if cfg.OfflineASR.Encoder == "" || cfg.OfflineASR.Encoder == "encoder.onnx" {
			// Use default model file name
			modelPath = filepath.Join(modelsDir, "model.int8.onnx")
		}
		recognizerConfig.ModelConfig.Paraformer.Model = modelPath
	case "whisper":
		// Whisper still uses separate Encoder and Decoder files
		recognizerConfig.ModelConfig.Whisper.Encoder = filepath.Join(modelsDir, cfg.OfflineASR.Encoder)
		recognizerConfig.ModelConfig.Whisper.Decoder = filepath.Join(modelsDir, cfg.OfflineASR.Decoder)
	case "transducer":
		// Transducer models use encoder, decoder, and joiner
		recognizerConfig.ModelConfig.Transducer.Encoder = filepath.Join(modelsDir, cfg.OfflineASR.Encoder)
		recognizerConfig.ModelConfig.Transducer.Decoder = filepath.Join(modelsDir, cfg.OfflineASR.Decoder)
		// Note: joiner should be added to config if needed
	}

	recognizerConfig.ModelConfig.Tokens = filepath.Join(modelsDir, cfg.OfflineASR.Tokens)
	recognizerConfig.ModelConfig.NumThreads = cfg.OfflineASR.NumThreads
	recognizerConfig.ModelConfig.Provider = "cpu"
	recognizerConfig.ModelConfig.Debug = 0
	recognizerConfig.ModelConfig.ModelType = cfg.OfflineASR.ModelType

	// 解码配置
	recognizerConfig.DecodingMethod = cfg.OfflineASR.DecodingMethod
	recognizerConfig.MaxActivePaths = cfg.OfflineASR.MaxActivePaths

	// 创建识别器
	recognizer := sherpa.NewOfflineRecognizer(&recognizerConfig)
	if recognizer == nil {
		log.Fatal("Failed to create offline recognizer")
	}

	log.Println("Offline ASR Manager initialized successfully")

	// 创建标点符号管理器
	punctMgr := NewPunctuationManager(cfg)

	return &OfflineASRManager{
		config:      cfg,
		recognizer:  recognizer,
		punctuation: punctMgr,
	}
}

// Recognize 识别音频
func (m *OfflineASRManager) Recognize(samples []float32, sampleRate int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt64(&m.stats.totalRequests, 1)

	// 创建流
	stream := sherpa.NewOfflineStream(m.recognizer)
	if stream == nil {
		atomic.AddInt64(&m.stats.failureCount, 1)
		return "", fmt.Errorf("failed to create stream")
	}
	defer sherpa.DeleteOfflineStream(stream)

	// 接受音频数据
	stream.AcceptWaveform(sampleRate, samples)

	// 解码
	m.recognizer.Decode(stream)

	// 获取结果
	result := stream.GetResult()
	if result == nil {
		atomic.AddInt64(&m.stats.failureCount, 1)
		return "", fmt.Errorf("failed to get recognition result")
	}

	atomic.AddInt64(&m.stats.successCount, 1)

	// 添加标点符号
	textWithPunct := m.punctuation.AddPunctuation(result.Text)

	return textWithPunct, nil
}

// RecognizeSegment 识别音频片段（用于说话者分离）
func (m *OfflineASRManager) RecognizeSegment(samples []float32, sampleRate int) (string, error) {
	return m.Recognize(samples, sampleRate)
}

// GetStats 获取统计信息
func (m *OfflineASRManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests": atomic.LoadInt64(&m.stats.totalRequests),
		"success_count":  atomic.LoadInt64(&m.stats.successCount),
		"failure_count":  atomic.LoadInt64(&m.stats.failureCount),
	}
}

// Close 关闭管理器
func (m *OfflineASRManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("Closing Offline ASR Manager...")

	// 关闭标点符号管理器
	if m.punctuation != nil {
		m.punctuation.Close()
	}

	// 删除识别器
	sherpa.DeleteOfflineRecognizer(m.recognizer)

	log.Println("Offline ASR Manager closed")
}
