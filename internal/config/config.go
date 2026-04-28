package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server             ServerConfig             `yaml:"server"`
	Admin              AdminConfig              `yaml:"admin"`
	Signature          SignatureConfig          `yaml:"signature"`
	StreamingASR       StreamingASRConfig       `yaml:"streaming_asr"`
	OfflineASR         OfflineASRConfig         `yaml:"offline_asr"`
	SpeakerDiarization SpeakerDiarizationConfig `yaml:"speaker_diarization"`
	VAD                VADConfig                `yaml:"vad"`
	Punctuation        PunctuationConfig        `yaml:"punctuation"`
	Concurrency        ConcurrencyConfig        `yaml:"concurrency"`
	Logging            LoggingConfig            `yaml:"logging"`
}

type AdminConfig struct {
	Password string `yaml:"password"`
}

type SignatureConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Secret         string `yaml:"secret"`
	MaxSkewSeconds int64  `yaml:"max_skew_seconds"`
}

type ServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	MaxConnections int    `yaml:"max_connections"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
}

type StreamingASRConfig struct {
	Enabled                 bool    `yaml:"enabled"`
	ModelType               string  `yaml:"model_type"`
	ModelsDir               string  `yaml:"models_dir"`
	Encoder                 string  `yaml:"encoder"`
	Decoder                 string  `yaml:"decoder"`
	Joiner                  string  `yaml:"joiner"`
	Tokens                  string  `yaml:"tokens"`
	NumThreads              int     `yaml:"num_threads"`
	SampleRate              int     `yaml:"sample_rate"`
	FeatureDim              int     `yaml:"feature_dim"`
	EnableEndpoint          bool    `yaml:"enable_endpoint"`
	Rule1MinTrailingSilence float32 `yaml:"rule1_min_trailing_silence"`
	Rule2MinTrailingSilence float32 `yaml:"rule2_min_trailing_silence"`
	Rule3MinUtteranceLength float32 `yaml:"rule3_min_utterance_length"`
}

type OfflineASRConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	ModelType               string `yaml:"model_type"`
	ModelsDir               string `yaml:"models_dir"`
	Encoder                 string `yaml:"encoder"`
	Decoder                 string `yaml:"decoder"`
	Tokens                  string `yaml:"tokens"`
	NumThreads              int    `yaml:"num_threads"`
	SampleRate              int    `yaml:"sample_rate"`
	DecodingMethod          string `yaml:"decoding_method"`
	MaxActivePaths          int    `yaml:"max_active_paths"`
	MaxFileSizeMB           int    `yaml:"max_file_size_mb"`           // 最大文件大小（MB）
	ChunkDurationSec        int    `yaml:"chunk_duration_sec"`         // 分块处理时长（秒）
	MaxConcurrency          int    `yaml:"max_concurrency"`            // 最大并发处理数
	MaxProcessingTimeoutMin int    `yaml:"max_processing_timeout_min"` // 最大处理超时时间（分钟）
}

type SpeakerDiarizationConfig struct {
	Enabled           bool             `yaml:"enabled"`
	ModelsDir         string           `yaml:"models_dir"`
	SegmentationModel string           `yaml:"segmentation_model"`
	EmbeddingModel    string           `yaml:"embedding_model"`
	Clustering        ClusteringConfig `yaml:"clustering"`
	NumThreads        int              `yaml:"num_threads"`
}

type ClusteringConfig struct {
	NumClusters int     `yaml:"num_clusters"`
	Threshold   float32 `yaml:"threshold"`
	MaxSpeakers int     `yaml:"max_speakers"` // 最大说话者数量限制
}

type VADConfig struct {
	Enabled            bool    `yaml:"enabled"`
	Model              string  `yaml:"model"`
	SampleRate         int     `yaml:"sample_rate"`
	MinSilenceDuration int     `yaml:"min_silence_duration"`
	MinSpeechDuration  int     `yaml:"min_speech_duration"`
	Threshold          float32 `yaml:"threshold"`
	WindowSize         int     `yaml:"window_size"`
	NumThreads         int     `yaml:"num_threads"`
}

type PunctuationConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ModelDir   string `yaml:"model_dir"`
	Model      string `yaml:"model"`
	NumThreads int    `yaml:"num_threads"`
}

type ConcurrencyConfig struct {
	MaxStreamingSessions int `yaml:"max_streaming_sessions"`
	MaxOfflineJobs       int `yaml:"max_offline_jobs"`
	WorkerPoolSize       int `yaml:"worker_pool_size"`
	QueueSize            int `yaml:"queue_size"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// LoadConfig 从文件加载配置
func LoadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	// 默认启用签名校验，并允许 5 分钟时间偏差。
	config.Signature.Enabled = true
	config.Signature.MaxSkewSeconds = 300
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Signature.Secret == "" {
		config.Signature.Secret = os.Getenv("API_SIGNATURE_SECRET")
	}

	if config.Signature.MaxSkewSeconds <= 0 {
		config.Signature.MaxSkewSeconds = 300
	}

	return &config, nil
}
