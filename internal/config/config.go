package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server             ServerConfig             `yaml:"server"`
	StreamingASR       StreamingASRConfig       `yaml:"streaming_asr"`
	OfflineASR         OfflineASRConfig         `yaml:"offline_asr"`
	SpeakerDiarization SpeakerDiarizationConfig `yaml:"speaker_diarization"`
	VAD                VADConfig                `yaml:"vad"`
	Punctuation        PunctuationConfig        `yaml:"punctuation"`
	Concurrency        ConcurrencyConfig        `yaml:"concurrency"`
	Logging            LoggingConfig            `yaml:"logging"`
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
	Enabled        bool   `yaml:"enabled"`
	ModelType      string `yaml:"model_type"`
	ModelsDir      string `yaml:"models_dir"`
	Encoder        string `yaml:"encoder"`
	Decoder        string `yaml:"decoder"`
	Tokens         string `yaml:"tokens"`
	NumThreads     int    `yaml:"num_threads"`
	SampleRate     int    `yaml:"sample_rate"`
	DecodingMethod string `yaml:"decoding_method"`
	MaxActivePaths int    `yaml:"max_active_paths"`
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
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
