package asr

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"

	"airecorder/internal/config"

	"github.com/google/uuid"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// StreamingASRSession 实时识别会话
type StreamingASRSession struct {
	ID          string
	Recognizer  *sherpa.OnlineRecognizer
	Stream      *sherpa.OnlineStream
	Punctuation *PunctuationManager
	mu          sync.Mutex
}

// ProcessAudio 处理音频数据并返回识别结果
func (s *StreamingASRSession) ProcessAudio(samples []float32) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 接受音频数据
	s.Stream.AcceptWaveform(16000, samples)

	// 解码
	for s.Recognizer.IsReady(s.Stream) {
		s.Recognizer.Decode(s.Stream)
	}

	// 获取结果
	result := s.Recognizer.GetResult(s.Stream)
	text := result.Text

	// 如果启用了标点符号且文本不为空，添加标点符号
	if text != "" && s.Punctuation != nil {
		text = s.Punctuation.AddPunctuation(text)
	}

	// 检查是否是端点
	isEndpoint := s.Recognizer.IsEndpoint(s.Stream)

	return text, isEndpoint, nil
}

// Reset 重置会话
func (s *StreamingASRSession) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Recognizer.Reset(s.Stream)
}

// StreamingASRManager 实时识别管理器
type StreamingASRManager struct {
	config      *config.Config
	recognizer  *sherpa.OnlineRecognizer
	punctuation *PunctuationManager
	sessions    map[string]*StreamingASRSession
	mu          sync.RWMutex
	stats       struct {
		activeSessions   int64
		totalSessions    int64
		totalAudioFrames int64
	}
}

// NewStreamingASRManager 创建实时识别管理器
func NewStreamingASRManager(cfg *config.Config) *StreamingASRManager {
	log.Println("Initializing Streaming ASR Manager...")

	// 构建模型路径
	modelsDir := cfg.StreamingASR.ModelsDir
	encoderPath := filepath.Join(modelsDir, cfg.StreamingASR.Encoder)
	decoderPath := filepath.Join(modelsDir, cfg.StreamingASR.Decoder)
	joinerPath := filepath.Join(modelsDir, cfg.StreamingASR.Joiner)
	tokensPath := filepath.Join(modelsDir, cfg.StreamingASR.Tokens)

	// 创建识别器配置
	recognizerConfig := sherpa.OnlineRecognizerConfig{}

	// 特征配置
	recognizerConfig.FeatConfig.SampleRate = cfg.StreamingASR.SampleRate
	recognizerConfig.FeatConfig.FeatureDim = cfg.StreamingASR.FeatureDim

	// 模型配置
	recognizerConfig.ModelConfig.Transducer.Encoder = encoderPath
	recognizerConfig.ModelConfig.Transducer.Decoder = decoderPath
	recognizerConfig.ModelConfig.Transducer.Joiner = joinerPath
	recognizerConfig.ModelConfig.Tokens = tokensPath
	recognizerConfig.ModelConfig.NumThreads = cfg.StreamingASR.NumThreads
	recognizerConfig.ModelConfig.Provider = "cpu"
	recognizerConfig.ModelConfig.Debug = 0
	recognizerConfig.ModelConfig.ModelType = cfg.StreamingASR.ModelType

	// 端点检测配置
	if cfg.StreamingASR.EnableEndpoint {
		recognizerConfig.Rule1MinTrailingSilence = cfg.StreamingASR.Rule1MinTrailingSilence
		recognizerConfig.Rule2MinTrailingSilence = cfg.StreamingASR.Rule2MinTrailingSilence
		recognizerConfig.Rule3MinUtteranceLength = cfg.StreamingASR.Rule3MinUtteranceLength
		recognizerConfig.EnableEndpoint = 1
	} else {
		recognizerConfig.EnableEndpoint = 0
	}

	// 创建识别器
	recognizer := sherpa.NewOnlineRecognizer(&recognizerConfig)
	if recognizer == nil {
		log.Fatal("Failed to create online recognizer")
	}

	log.Println("Streaming ASR Manager initialized successfully")

	// 创建标点符号管理器
	punctMgr := NewPunctuationManager(cfg)

	return &StreamingASRManager{
		config:      cfg,
		recognizer:  recognizer,
		punctuation: punctMgr,
		sessions:    make(map[string]*StreamingASRSession),
	}
}

// CreateSession 创建新的识别会话
func (m *StreamingASRManager) CreateSession() (*StreamingASRSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查并发限制
	if int(atomic.LoadInt64(&m.stats.activeSessions)) >= m.config.Concurrency.MaxStreamingSessions {
		return nil, fmt.Errorf("maximum concurrent sessions reached")
	}

	sessionID := uuid.New().String()

	// 创建流
	stream := sherpa.NewOnlineStream(m.recognizer)
	if stream == nil {
		return nil, fmt.Errorf("failed to create stream")
	}

	session := &StreamingASRSession{
		ID:          sessionID,
		Recognizer:  m.recognizer,
		Stream:      stream,
		Punctuation: m.punctuation,
	}

	m.sessions[sessionID] = session

	atomic.AddInt64(&m.stats.activeSessions, 1)
	atomic.AddInt64(&m.stats.totalSessions, 1)

	log.Printf("Created streaming session: %s (active: %d)", sessionID, atomic.LoadInt64(&m.stats.activeSessions))

	return session, nil
}

// CloseSession 关闭识别会话
func (m *StreamingASRManager) CloseSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[sessionID]; exists {
		sherpa.DeleteOnlineStream(session.Stream)
		delete(m.sessions, sessionID)
		atomic.AddInt64(&m.stats.activeSessions, -1)
		log.Printf("Closed streaming session: %s (active: %d)", sessionID, atomic.LoadInt64(&m.stats.activeSessions))
	}
}

// GetStats 获取统计信息
func (m *StreamingASRManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_sessions":    atomic.LoadInt64(&m.stats.activeSessions),
		"total_sessions":     atomic.LoadInt64(&m.stats.totalSessions),
		"total_audio_frames": atomic.LoadInt64(&m.stats.totalAudioFrames),
	}
}

// Close 关闭管理器
func (m *StreamingASRManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("Closing Streaming ASR Manager...")

	// 关闭所有会话
	for sessionID, session := range m.sessions {
		sherpa.DeleteOnlineStream(session.Stream)
		delete(m.sessions, sessionID)
	}

	// 关闭标点符号管理器
	if m.punctuation != nil {
		m.punctuation.Close()
	}

	// 删除识别器
	sherpa.DeleteOnlineRecognizer(m.recognizer)

	log.Println("Streaming ASR Manager closed")
}
