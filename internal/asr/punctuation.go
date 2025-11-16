package asr

import (
	"log"
	"path/filepath"
	"sync"

	"airecorder/internal/config"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// PunctuationManager 标点符号管理器
type PunctuationManager struct {
	config      *config.Config
	punctuation *sherpa.OfflinePunctuation
	mu          sync.Mutex
	enabled     bool
}

// NewPunctuationManager 创建标点符号管理器
func NewPunctuationManager(cfg *config.Config) *PunctuationManager {
	if !cfg.Punctuation.Enabled {
		log.Println("Punctuation is disabled")
		return &PunctuationManager{
			config:  cfg,
			enabled: false,
		}
	}

	log.Println("Initializing Punctuation Manager...")

	// 构建模型路径
	modelPath := filepath.Join(cfg.Punctuation.ModelDir, cfg.Punctuation.Model)

	// 创建标点符号配置
	punctConfig := sherpa.OfflinePunctuationConfig{}
	punctConfig.Model.CtTransformer = modelPath
	// Note: CGO type issue - using literal constant instead of variable
	punctConfig.Model.NumThreads = 1
	punctConfig.Model.Debug = 0
	punctConfig.Model.Provider = "cpu"

	// 创建标点符号处理器
	punctuation := sherpa.NewOfflinePunctuation(&punctConfig)
	if punctuation == nil {
		log.Println("Warning: Failed to create punctuation processor, will return text without punctuation")
		return &PunctuationManager{
			config:  cfg,
			enabled: false,
		}
	}

	log.Println("Punctuation Manager initialized successfully")

	return &PunctuationManager{
		config:      cfg,
		punctuation: punctuation,
		enabled:     true,
	}
}

// AddPunctuation 为文本添加标点符号
func (m *PunctuationManager) AddPunctuation(text string) string {
	if !m.enabled || text == "" {
		return text
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用 sherpa-onnx 的标点符号添加功能
	result := m.punctuation.AddPunct(text)

	return result
}

// IsEnabled 返回标点符号功能是否启用
func (m *PunctuationManager) IsEnabled() bool {
	return m.enabled
}

// Close 关闭管理器
func (m *PunctuationManager) Close() {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("Closing Punctuation Manager...")

	// 删除标点符号处理器
	if m.punctuation != nil {
		sherpa.DeleteOfflinePunc(m.punctuation)
	}

	log.Println("Punctuation Manager closed")
}
