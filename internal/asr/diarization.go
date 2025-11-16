package asr

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"airecorder/internal/config"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// DiarizationSegment 说话者分离片段
type DiarizationSegment struct {
	Start   float32
	End     float32
	Speaker int
	Text    string
}

// DiarizationManager 说话者分离管理器
type DiarizationManager struct {
	config      *config.Config
	diarization *sherpa.OfflineSpeakerDiarization
	mu          sync.Mutex
}

// NewDiarizationManager 创建说话者分离管理器
func NewDiarizationManager(cfg *config.Config) *DiarizationManager {
	log.Println("Initializing Speaker Diarization Manager...")

	modelsDir := cfg.SpeakerDiarization.ModelsDir

	diarizationConfig := sherpa.OfflineSpeakerDiarizationConfig{}

	// 分割模型配置
	diarizationConfig.Segmentation.Pyannote.Model = filepath.Join(modelsDir, cfg.SpeakerDiarization.SegmentationModel)

	// 说话者嵌入模型配置
	diarizationConfig.Embedding.Model = filepath.Join(modelsDir, cfg.SpeakerDiarization.EmbeddingModel)

	// 聚类配置
	if cfg.SpeakerDiarization.Clustering.NumClusters > 0 {
		diarizationConfig.Clustering.NumClusters = cfg.SpeakerDiarization.Clustering.NumClusters
	} else {
		diarizationConfig.Clustering.Threshold = cfg.SpeakerDiarization.Clustering.Threshold
	}

	diarizationConfig.Segmentation.NumThreads = cfg.SpeakerDiarization.NumThreads
	diarizationConfig.Embedding.NumThreads = cfg.SpeakerDiarization.NumThreads

	// 创建说话者分离器
	diarization := sherpa.NewOfflineSpeakerDiarization(&diarizationConfig)
	if diarization == nil {
		log.Fatal("Failed to create speaker diarization")
	}

	log.Println("Speaker Diarization Manager initialized successfully")

	return &DiarizationManager{
		config:      cfg,
		diarization: diarization,
	}
}

// Process 处理音频并返回说话者分离片段（不包含文本）
func (m *DiarizationManager) Process(samples []float32, sampleRate int) ([]DiarizationSegment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查采样率
	expectedSampleRate := m.diarization.SampleRate()
	if sampleRate != expectedSampleRate {
		return nil, fmt.Errorf("sample rate mismatch: expected %d, got %d", expectedSampleRate, sampleRate)
	}

	// 处理音频
	segments := m.diarization.Process(samples)

	// 转换为我们的格式
	result := make([]DiarizationSegment, len(segments))
	for i, seg := range segments {
		result[i] = DiarizationSegment{
			Start:   seg.Start,
			End:     seg.End,
			Speaker: seg.Speaker,
		}
	}

	// 先进行基于时间连续性的合并（合并相邻的同一说话者片段）
	result = m.mergeAdjacentSegments(result)

	// 再进行智能说话者合并（减少说话者数量）
	maxSpeakers := m.config.SpeakerDiarization.Clustering.MaxSpeakers
	if maxSpeakers > 0 {
		result = m.mergeSpeakersIfNeeded(result, maxSpeakers)
	}

	return result, nil
}

// mergeAdjacentSegments 合并相邻的同一说话者片段
func (m *DiarizationManager) mergeAdjacentSegments(segments []DiarizationSegment) []DiarizationSegment {
	if len(segments) <= 1 {
		return segments
	}

	merged := make([]DiarizationSegment, 0, len(segments))
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		next := segments[i]

		// 如果是同一个说话者，且时间间隔很小（小于0.5秒），则合并
		if current.Speaker == next.Speaker && (next.Start-current.End) < 0.5 {
			current.End = next.End
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	if len(merged) < len(segments) {
		log.Printf("Merged adjacent segments: %d -> %d", len(segments), len(merged))
	}

	return merged
}

// mergeSpeakersIfNeeded 如果说话者数量超过限制，则合并相似的说话者
func (m *DiarizationManager) mergeSpeakersIfNeeded(segments []DiarizationSegment, maxSpeakers int) []DiarizationSegment {
	// 统计唯一说话者
	speakerSet := make(map[int]bool)
	for _, seg := range segments {
		speakerSet[seg.Speaker] = true
	}

	numSpeakers := len(speakerSet)
	if numSpeakers <= maxSpeakers {
		return segments // 不需要合并
	}

	log.Printf("Warning: Detected %d speakers, exceeding maximum %d. Merging speakers...", numSpeakers, maxSpeakers)

	// 统计每个说话者的信息
	type speakerStats struct {
		id            int
		totalDuration float32
		segmentCount  int
		avgDuration   float32
	}

	statsMap := make(map[int]*speakerStats)
	for _, seg := range segments {
		if _, exists := statsMap[seg.Speaker]; !exists {
			statsMap[seg.Speaker] = &speakerStats{id: seg.Speaker}
		}
		stats := statsMap[seg.Speaker]
		stats.totalDuration += seg.End - seg.Start
		stats.segmentCount++
	}

	// 计算平均时长
	statsList := make([]*speakerStats, 0, len(statsMap))
	for _, stats := range statsMap {
		stats.avgDuration = stats.totalDuration / float32(stats.segmentCount)
		statsList = append(statsList, stats)
	}

	// 按总时长降序排序（时长长的说话者优先保留）
	for i := 0; i < len(statsList)-1; i++ {
		for j := i + 1; j < len(statsList); j++ {
			if statsList[j].totalDuration > statsList[i].totalDuration {
				statsList[i], statsList[j] = statsList[j], statsList[i]
			}
		}
	}

	// 建立说话者映射：保留前maxSpeakers个说话者，其他映射到这些保留的说话者
	speakerMapping := make(map[int]int)

	// 主要说话者（保留）
	mainSpeakers := make([]int, 0, maxSpeakers)
	for i := 0; i < len(statsList) && i < maxSpeakers; i++ {
		speakerId := statsList[i].id
		speakerMapping[speakerId] = len(mainSpeakers) // 重新编号为 0, 1, 2, ...
		mainSpeakers = append(mainSpeakers, speakerId)
		log.Printf("  Keep Speaker %d -> %d (duration: %.2fs, segments: %d)",
			speakerId, len(mainSpeakers)-1, statsList[i].totalDuration, statsList[i].segmentCount)
	}

	// 次要说话者（需要合并）
	for i := maxSpeakers; i < len(statsList); i++ {
		// 将次要说话者轮流分配到主要说话者
		targetIdx := i % maxSpeakers
		speakerMapping[statsList[i].id] = targetIdx
		log.Printf("  Merge Speaker %d -> Speaker %d (duration: %.2fs, segments: %d)",
			statsList[i].id, targetIdx, statsList[i].totalDuration, statsList[i].segmentCount)
	}

	// 应用映射
	for i := range segments {
		segments[i].Speaker = speakerMapping[segments[i].Speaker]
	}

	// 合并后再次合并相邻片段（因为映射后可能产生新的相邻同说话者片段）
	segments = m.mergeAdjacentSegments(segments)

	log.Printf("Successfully merged speakers from %d to %d (final segments: %d)", numSpeakers, maxSpeakers, len(segments))
	return segments
}

// ProcessWithASR 处理音频并结合 ASR 识别每个片段
func (m *DiarizationManager) ProcessWithASR(samples []float32, sampleRate int, asrManager *OfflineASRManager) ([]DiarizationSegment, error) {
	// 先进行说话者分离
	segments, err := m.Process(samples, sampleRate)
	if err != nil {
		return nil, err
	}

	// 对每个片段进行语音识别
	for i := range segments {
		seg := &segments[i]

		// 计算片段的样本索引
		startIdx := int(seg.Start * float32(sampleRate))
		endIdx := int(seg.End * float32(sampleRate))

		// 确保索引在有效范围内
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx > len(samples) {
			endIdx = len(samples)
		}

		// 提取片段音频
		segmentSamples := samples[startIdx:endIdx]

		// 识别该片段
		text, err := asrManager.RecognizeSegment(segmentSamples, sampleRate)
		if err != nil {
			log.Printf("Warning: failed to recognize segment %d: %v", i, err)
			seg.Text = ""
		} else {
			seg.Text = text
		}
	}

	return segments, nil
}

// Close 关闭管理器
func (m *DiarizationManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Println("Closing Speaker Diarization Manager...")

	// 删除说话者分离器
	sherpa.DeleteOfflineSpeakerDiarization(m.diarization)

	log.Println("Speaker Diarization Manager closed")
}
