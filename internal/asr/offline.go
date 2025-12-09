package asr

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

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

// RecognizeChunked 分块识别长音频
func (m *OfflineASRManager) RecognizeChunked(samples []float32, sampleRate int) (string, error) {
	// 获取分块时长配置（默认60秒，提高处理效率）
	chunkDurationSec := m.config.OfflineASR.ChunkDurationSec
	if chunkDurationSec <= 0 {
		chunkDurationSec = 60 // 使用60秒的块，大幅提高效率
	}

	chunkSize := sampleRate * chunkDurationSec
	totalSamples := len(samples)
	totalDuration := float64(totalSamples) / float64(sampleRate)

	// 如果音频短于分块大小，直接识别
	if totalSamples <= chunkSize {
		return m.Recognize(samples, sampleRate)
	}

	log.Printf("[ChunkedASR] Starting chunked recognition: total_duration=%.2fs, chunk_duration=%ds, estimated_chunks=%d",
		totalDuration, chunkDurationSec, (totalSamples+chunkSize-1)/chunkSize)

	// 动态计算超时时间：基础时间 + 音频时长的3倍（考虑处理开销）
	// 最小30分钟，最大从配置读取（默认120分钟）
	maxTimeoutMin := m.config.OfflineASR.MaxProcessingTimeoutMin
	if maxTimeoutMin <= 0 {
		maxTimeoutMin = 120 // 默认120分钟
	}

	baseTimeout := 30 * time.Minute
	processingTimeout := time.Duration(totalDuration*3) * time.Second
	totalTimeout := baseTimeout + processingTimeout
	maxTimeout := time.Duration(maxTimeoutMin) * time.Minute
	if totalTimeout > maxTimeout {
		totalTimeout = maxTimeout
	}

	log.Printf("[ChunkedASR] Setting processing timeout: %.2f minutes (audio: %.2f min, max: %d min)",
		totalTimeout.Minutes(), totalDuration/60, maxTimeoutMin)
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	// 使用goroutine池并行处理，提高效率
	maxWorkers := 4 // 默认使用4个worker，加快处理速度
	if m.config.OfflineASR.MaxConcurrency > 0 {
		maxWorkers = m.config.OfflineASR.MaxConcurrency
	}

	type chunkResult struct {
		index int
		text  string
		err   error
	}

	// 计算总块数
	numChunks := (totalSamples + chunkSize - 1) / chunkSize
	results := make([]string, numChunks)
	resultChan := make(chan chunkResult, numChunks)
	workChan := make(chan int, numChunks)

	// 启动worker池
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for chunkIndex := range workChan {
				// 检查是否超时
				select {
				case <-ctx.Done():
					log.Printf("[ChunkedASR] Worker %d stopped due to timeout", workerID)
					resultChan <- chunkResult{index: chunkIndex, err: fmt.Errorf("processing timeout")}
					return
				default:
				}

				offset := chunkIndex * chunkSize
				end := offset + chunkSize
				if end > totalSamples {
					end = totalSamples
				}

				chunkDur := float64(end-offset) / float64(sampleRate)
				progress := float64(chunkIndex+1) / float64(numChunks) * 100

				if chunkIndex%5 == 0 { // 每5个块输出一次日志
					log.Printf("[ChunkedASR] Worker %d processing chunk %d/%d: offset=%.2fs, duration=%.2fs, progress=%.1f%%",
						workerID, chunkIndex+1, numChunks, float64(offset)/float64(sampleRate), chunkDur, progress)
				}

				// 提取当前块
				chunk := samples[offset:end]

				// 识别当前块（每个worker有自己的锁，减少竞争）
				text, err := m.recognizeChunkWithCleanup(chunk, sampleRate, chunkIndex+1)
				resultChan <- chunkResult{index: chunkIndex, text: text, err: err}
			}
		}(w)
	}

	// 分发所有任务
	go func() {
		for i := 0; i < numChunks; i++ {
			workChan <- i
		}
		close(workChan)
	}()

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	failedChunks := 0
	for result := range resultChan {
		if result.err != nil {
			log.Printf("[ChunkedASR] Warning: chunk %d failed: %v", result.index+1, result.err)
			failedChunks++
		} else {
			results[result.index] = result.text
		}
	}

	// 合并所有结果
	var fullText string
	for i, text := range results {
		if text != "" {
			if fullText != "" {
				fullText += " "
			}
			fullText += text
		} else if i < len(results)-1 { // 不是最后一块但为空，可能失败了
			log.Printf("[ChunkedASR] Warning: chunk %d has no text", i+1)
		}
	}

	log.Printf("[ChunkedASR] Completed: total_chunks=%d, failed_chunks=%d, result_length=%d chars",
		numChunks, failedChunks, len(fullText))

	if fullText == "" && failedChunks > 0 {
		return "", fmt.Errorf("all chunks failed to recognize")
	}

	return fullText, nil
}

// recognizeChunkWithCleanup 识别单个块并确保资源清理
func (m *OfflineASRManager) recognizeChunkWithCleanup(samples []float32, sampleRate int, chunkID int) (string, error) {
	// 不使用全局锁，让多个块可以并发处理（如果需要的话）
	// 但由于 sherpa-onnx 的线程安全性，这里还是用锁
	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建流
	stream := sherpa.NewOfflineStream(m.recognizer)
	if stream == nil {
		return "", fmt.Errorf("failed to create stream for chunk %d", chunkID)
	}
	// 确保流被释放
	defer sherpa.DeleteOfflineStream(stream)

	// 接受音频数据
	stream.AcceptWaveform(sampleRate, samples)

	// 解码
	m.recognizer.Decode(stream)

	// 获取结果
	result := stream.GetResult()
	if result == nil {
		return "", fmt.Errorf("failed to get recognition result for chunk %d", chunkID)
	}

	// 添加标点符号
	textWithPunct := m.punctuation.AddPunctuation(result.Text)

	return textWithPunct, nil
}

// GetMaxFileSizeMB 获取最大文件大小配置（MB）
func (m *OfflineASRManager) GetMaxFileSizeMB() int {
	if m.config.OfflineASR.MaxFileSizeMB > 0 {
		return m.config.OfflineASR.MaxFileSizeMB
	}
	return 50 // 默认50MB
}

// GetChunkDurationSec 获取分块时长配置（秒）
func (m *OfflineASRManager) GetChunkDurationSec() int {
	if m.config.OfflineASR.ChunkDurationSec > 0 {
		return m.config.OfflineASR.ChunkDurationSec
	}
	return 30 // 默认30秒
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
