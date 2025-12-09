package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"

	"airecorder/internal/asr"
	"airecorder/internal/audio"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应该检查 origin
	},
}

// StreamingASRRequest WebSocket 消息格式
type StreamingASRMessage struct {
	Type       string `json:"type"`  // "audio" 或 "control"
	Audio      string `json:"audio"` // Base64 编码的音频数据
	SampleRate int    `json:"sample_rate,omitempty"`
	Command    string `json:"command,omitempty"` // "start", "stop", "reset"
}

// StreamingASRResponse WebSocket 响应格式
type StreamingASRResponse struct {
	Type       string `json:"type"` // "result", "partial", "error"
	Text       string `json:"text"`
	IsEndpoint bool   `json:"is_endpoint,omitempty"`
	Segment    int    `json:"segment,omitempty"`
	Error      string `json:"error,omitempty"`
}

// HandleStreamingASR 处理实时语音识别 WebSocket 连接
func HandleStreamingASR(c *gin.Context, manager *asr.StreamingASRManager) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 创建会话
	session, err := manager.CreateSession()
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		conn.WriteJSON(StreamingASRResponse{
			Type:  "error",
			Error: "Failed to create session: " + err.Error(),
		})
		return
	}
	defer manager.CloseSession(session.ID)

	log.Printf("Streaming ASR session %s started", session.ID)

	// 发送欢迎消息
	conn.WriteJSON(StreamingASRResponse{
		Type: "result",
		Text: "Connected. Ready to receive audio.",
	})

	segmentIdx := 0
	lastText := ""

	// 读取消息循环
	for {
		var msg StreamingASRMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "audio":
			// 解码音频数据
			audioData, err := base64.StdEncoding.DecodeString(msg.Audio)
			if err != nil {
				conn.WriteJSON(StreamingASRResponse{
					Type:  "error",
					Error: "Invalid audio data: " + err.Error(),
				})
				continue
			}

			// 将字节数据转换为 float32 样本
			samples := bytesToFloat32(audioData)

			// 处理音频
			result, isEndpoint, err := session.ProcessAudio(samples)
			if err != nil {
				conn.WriteJSON(StreamingASRResponse{
					Type:  "error",
					Error: "Processing error: " + err.Error(),
				})
				continue
			}

			// 如果有新的识别结果，发送回客户端
			if result != "" && result != lastText {
				lastText = result
				conn.WriteJSON(StreamingASRResponse{
					Type:       "partial",
					Text:       result,
					IsEndpoint: isEndpoint,
					Segment:    segmentIdx,
				})
			}

			// 如果检测到端点，重置
			if isEndpoint {
				if result != "" {
					segmentIdx++
					conn.WriteJSON(StreamingASRResponse{
						Type:       "result",
						Text:       result,
						IsEndpoint: true,
						Segment:    segmentIdx,
					})
				}
				session.Reset()
				lastText = ""
			}

		case "control":
			switch msg.Command {
			case "reset":
				session.Reset()
				lastText = ""
				conn.WriteJSON(StreamingASRResponse{
					Type: "result",
					Text: "Session reset",
				})
			case "stop":
				conn.WriteJSON(StreamingASRResponse{
					Type: "result",
					Text: "Session stopped",
				})
				return
			}
		}
	}

	log.Printf("Streaming ASR session %s ended", session.ID)
}

// OfflineASRRequest 离线识别请求格式
type OfflineASRRequest struct {
	Audio             string `json:"audio" form:"audio"`                           // Base64 编码的音频数据
	SampleRate        int    `json:"sample_rate" form:"sample_rate"`               // 采样率，默认 16000
	EnableDiarization bool   `json:"enable_diarization" form:"enable_diarization"` // 是否启用说话者分离
}

// OfflineASRResponse 离线识别响应格式
type OfflineASRResponse struct {
	Text     string               `json:"text"`
	Segments []DiarizationSegment `json:"segments,omitempty"`
	Duration float32              `json:"duration,omitempty"`
	Error    string               `json:"error,omitempty"`
}

// DiarizationSegment 说话者分离片段
type DiarizationSegment struct {
	Start   float32 `json:"start"`   // 开始时间（秒）
	End     float32 `json:"end"`     // 结束时间（秒）
	Speaker int     `json:"speaker"` // 说话者 ID
	Text    string  `json:"text"`    // 识别文本
}

// HandleOfflineASR 处理离线语音识别
func HandleOfflineASR(c *gin.Context, asrManager *asr.OfflineASRManager, diarizationMgr *asr.DiarizationManager) {
	HandleOfflineASRWithQueue(c, asrManager, diarizationMgr, nil)
}

// HandleOfflineASRWithQueue 处理离线语音识别（带队列支持）
func HandleOfflineASRWithQueue(c *gin.Context, asrManager *asr.OfflineASRManager, diarizationMgr *asr.DiarizationManager, taskQueue *asr.TaskQueue) {
	var req OfflineASRRequest
	var audioData []byte
	var fileSize int64

	// 支持 JSON 和 Form 数据
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, OfflineASRResponse{
				Error: "Invalid request: " + err.Error(),
			})
			return
		}

		// 解码 Base64 音频数据
		var err error
		audioData, err = base64.StdEncoding.DecodeString(req.Audio)
		if err != nil {
			c.JSON(http.StatusBadRequest, OfflineASRResponse{
				Error: "Invalid audio data: " + err.Error(),
			})
			return
		}
		fileSize = int64(len(audioData))
	} else {
		// 处理文件上传
		file, err := c.FormFile("audio_file")
		if err == nil {
			fileSize = file.Size

			// 检查文件大小（从配置获取，默认50MB）
			maxFileSizeMB := asrManager.GetMaxFileSizeMB()
			maxFileSize := int64(maxFileSizeMB) << 20
			if fileSize > maxFileSize {
				c.JSON(http.StatusRequestEntityTooLarge, OfflineASRResponse{
					Error: fmt.Sprintf("File size (%d MB) exceeds maximum allowed size (%d MB)", fileSize>>20, maxFileSizeMB),
				})
				return
			}

			// 读取文件
			f, err := file.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Failed to open file: " + err.Error(),
				})
				return
			}
			defer f.Close()

			audioData, err = io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Failed to read file: " + err.Error(),
				})
				return
			}
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Invalid request: " + err.Error(),
				})
				return
			}

			// 解码 Base64 音频数据
			audioData, err = base64.StdEncoding.DecodeString(req.Audio)
			if err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Invalid audio data: " + err.Error(),
				})
				return
			}
			fileSize = int64(len(audioData))
		}
	}

	log.Printf("Processing audio file: size=%d bytes (%.2f MB)", fileSize, float64(fileSize)/(1024*1024))

	// 使用音频转换器自动检测和转换格式
	converter := audio.NewAudioConverter()
	samples, sampleRate, convertErr := converter.ConvertToSamples(audioData)
	if convertErr != nil {
		c.JSON(http.StatusBadRequest, OfflineASRResponse{
			Error: "Audio format conversion failed: " + convertErr.Error(),
		})
		return
	}

	log.Printf("Audio converted successfully: %d samples at %d Hz", len(samples), sampleRate)

	// 如果请求中指定了采样率，使用转换后的实际采样率
	if req.SampleRate == 0 {
		req.SampleRate = sampleRate
	}

	// 计算音频时长
	audioDuration := float32(len(samples)) / float32(req.SampleRate)

	// 判断是否使用队列处理（音频时长超过2分钟且队列可用）
	useQueue := taskQueue != nil && audioDuration > 120.0 // 2分钟

	if useQueue {
		log.Printf("Audio duration (%.2fs) > 120s, using task queue", audioDuration)

		// 创建任务
		enableDiar := diarizationMgr != nil
		task := asr.NewASRTask(samples, req.SampleRate, diarizationMgr, enableDiar)

		// 提交任务
		if err := taskQueue.Submit(task); err != nil {
			c.JSON(http.StatusServiceUnavailable, OfflineASRResponse{
				Error: "Task queue full: " + err.Error(),
			})
			return
		}

		// 等待任务完成（对调用方无感，阻塞等待）
		result := task.Wait()

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, OfflineASRResponse{
				Error: "Recognition error: " + result.Error.Error(),
			})
			return
		}

		// 返回结果
		if enableDiar {
			diarSegments := make([]DiarizationSegment, len(result.Segments))
			for i, seg := range result.Segments {
				diarSegments[i] = DiarizationSegment{
					Start:   seg.Start,
					End:     seg.End,
					Speaker: seg.Speaker,
					Text:    seg.Text,
				}
			}

			c.JSON(http.StatusOK, OfflineASRResponse{
				Text:     result.Text,
				Segments: diarSegments,
				Duration: result.Duration,
			})
		} else {
			c.JSON(http.StatusOK, OfflineASRResponse{
				Text:     result.Text,
				Duration: result.Duration,
			})
		}
		return
	}

	// 直接处理（不使用队列）
	if diarizationMgr != nil {
		segments, err := diarizationMgr.ProcessWithASR(samples, req.SampleRate, asrManager)
		if err != nil {
			c.JSON(http.StatusInternalServerError, OfflineASRResponse{
				Error: "Diarization error: " + err.Error(),
			})
			return
		}

		// 组合所有文本
		fullText := ""
		diarSegments := make([]DiarizationSegment, len(segments))
		for i, seg := range segments {
			fullText += seg.Text + " "
			diarSegments[i] = DiarizationSegment{
				Start:   seg.Start,
				End:     seg.End,
				Speaker: seg.Speaker,
				Text:    seg.Text,
			}
		}

		c.JSON(http.StatusOK, OfflineASRResponse{
			Text:     fullText,
			Segments: diarSegments,
			Duration: audioDuration,
		})
		return
	}

	// 普通识别（不带说话者分离）
	chunkDurationSec := asrManager.GetChunkDurationSec()

	var text string
	var err error

	if audioDuration > float32(chunkDurationSec) {
		log.Printf("Audio duration (%.2fs) exceeds chunk duration (%ds), using chunked processing", audioDuration, chunkDurationSec)
		text, err = asrManager.RecognizeChunked(samples, req.SampleRate)
	} else {
		text, err = asrManager.Recognize(samples, req.SampleRate)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, OfflineASRResponse{
			Error: "Recognition error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, OfflineASRResponse{
		Text:     text,
		Duration: audioDuration,
	})
}

// HandleDiarization 处理独立的说话者分离请求
func HandleDiarization(c *gin.Context, manager *asr.DiarizationManager) {
	var req OfflineASRRequest
	var audioData []byte

	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request: " + err.Error(),
			})
			return
		}

		// 解码音频数据
		var err error
		audioData, err = base64.StdEncoding.DecodeString(req.Audio)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid audio data: " + err.Error(),
			})
			return
		}
	} else {
		// 处理文件上传
		file, err := c.FormFile("audio_file")
		if err == nil {
			f, err := file.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Failed to open file: " + err.Error(),
				})
				return
			}
			defer f.Close()

			audioData, err = io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Failed to read file: " + err.Error(),
				})
				return
			}
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request: " + err.Error(),
				})
				return
			}

			audioData, err = base64.StdEncoding.DecodeString(req.Audio)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid audio data: " + err.Error(),
				})
				return
			}
		}
	}

	// 使用音频转换器转换格式
	converter := audio.NewAudioConverter()
	samples, sampleRate, err := converter.ConvertToSamples(audioData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Audio format conversion failed: " + err.Error(),
		})
		return
	}

	if req.SampleRate == 0 {
		req.SampleRate = sampleRate
	}

	segments, err := manager.Process(samples, req.SampleRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Diarization error: " + err.Error(),
		})
		return
	}

	diarSegments := make([]DiarizationSegment, len(segments))
	for i, seg := range segments {
		diarSegments[i] = DiarizationSegment{
			Start:   seg.Start,
			End:     seg.End,
			Speaker: seg.Speaker,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"segments": diarSegments,
		"duration": float32(len(samples)) / float32(req.SampleRate),
	})
}

// HandleStats 处理统计信息请求
func HandleStats(c *gin.Context, streamingMgr *asr.StreamingASRManager, offlineMgr *asr.OfflineASRManager) {
	stats := gin.H{}

	if streamingMgr != nil {
		stats["streaming"] = streamingMgr.GetStats()
	}

	if offlineMgr != nil {
		stats["offline"] = offlineMgr.GetStats()
	}

	c.JSON(http.StatusOK, stats)
}

// bytesToFloat32 将字节数组转换为 float32 样本数组
func bytesToFloat32(data []byte) []float32 {
	if len(data)%2 != 0 {
		log.Printf("Warning: audio data length is not even, truncating last byte")
		data = data[:len(data)-1]
	}

	numSamples := len(data) / 2
	samples := make([]float32, numSamples)

	for i := 0; i < numSamples; i++ {
		// 小端序：低字节在前
		low := int16(data[2*i])
		high := int16(data[2*i+1])
		s16 := (high << 8) | low
		samples[i] = float32(s16) / 32768.0
	}

	return samples
}
