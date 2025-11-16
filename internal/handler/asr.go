package handler

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"

	"airecorder/internal/asr"

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
	var req OfflineASRRequest

	// 支持 JSON 和 Form 数据
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, OfflineASRResponse{
				Error: "Invalid request: " + err.Error(),
			})
			return
		}
	} else {
		// 处理文件上传
		file, err := c.FormFile("audio_file")
		if err == nil {
			// 读取文件
			f, err := file.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Failed to open file: " + err.Error(),
				})
				return
			}
			defer f.Close()

			audioBytes, err := io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Failed to read file: " + err.Error(),
				})
				return
			}

			req.Audio = base64.StdEncoding.EncodeToString(audioBytes)
			req.SampleRate = 16000 // 默认采样率
		} else {
			if err := c.ShouldBind(&req); err != nil {
				c.JSON(http.StatusBadRequest, OfflineASRResponse{
					Error: "Invalid request: " + err.Error(),
				})
				return
			}
		}
	}

	// 解码音频数据
	audioData, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		c.JSON(http.StatusBadRequest, OfflineASRResponse{
			Error: "Invalid audio data: " + err.Error(),
		})
		return
	}

	samples := bytesToFloat32(audioData)
	if req.SampleRate == 0 {
		req.SampleRate = 16000
	}

	// 如果 diarizationMgr 不为 nil，说明调用的是 diarization 端点，应该启用说话者分离
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

		duration := float32(len(samples)) / float32(req.SampleRate)

		c.JSON(http.StatusOK, OfflineASRResponse{
			Text:     fullText,
			Segments: diarSegments,
			Duration: duration,
		})
		return
	}

	// 普通识别（不带说话者分离）
	text, err := asrManager.Recognize(samples, req.SampleRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OfflineASRResponse{
			Error: "Recognition error: " + err.Error(),
		})
		return
	}

	duration := float32(len(samples)) / float32(req.SampleRate)

	c.JSON(http.StatusOK, OfflineASRResponse{
		Text:     text,
		Duration: duration,
	})
}

// HandleDiarization 处理独立的说话者分离请求
func HandleDiarization(c *gin.Context, manager *asr.DiarizationManager) {
	var req OfflineASRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// 解码音频数据
	audioData, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid audio data: " + err.Error(),
		})
		return
	}

	samples := bytesToFloat32(audioData)
	if req.SampleRate == 0 {
		req.SampleRate = 16000
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
