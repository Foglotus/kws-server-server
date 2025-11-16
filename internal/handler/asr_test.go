package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"airecorder/internal/asr"
	"airecorder/internal/config"

	"github.com/gin-gonic/gin"
)

func getTestAudioPath(filename string) string {
	// 尝试从环境变量获取项目根目录
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return filepath.Join(root, filename)
	}
	// 默认相对于当前工作目录
	return filepath.Join("../..", filename)
}

func setupTestRouter(asrMgr *asr.OfflineASRManager, diarizationMgr *asr.DiarizationManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// 离线识别接口
	router.POST("/api/v1/asr/offline", func(c *gin.Context) {
		HandleOfflineASR(c, asrMgr, diarizationMgr)
	})

	return router
}

func loadTestAudio(t *testing.T, filename string) string {
	// 尝试加载音频文件
	samples, sampleRate, err := asr.LoadAudioFile(filename, 16000)
	if err != nil {
		t.Skipf("Skipping test (audio file not available): %v", err)
		return ""
	}

	// 将samples转换为字节数组
	audioBytes := make([]byte, len(samples)*4)
	for i, sample := range samples {
		// 转换为16位PCM
		intSample := int16(sample * 32767.0)
		audioBytes[i*2] = byte(intSample & 0xff)
		audioBytes[i*2+1] = byte((intSample >> 8) & 0xff)
	}

	t.Logf("Loaded audio: %d samples at %d Hz", len(samples), sampleRate)
	return base64.StdEncoding.EncodeToString(audioBytes[:len(samples)*2])
}

func TestOfflineASRAPI(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建管理器
	asrMgr := asr.NewOfflineASRManager(cfg)
	if asrMgr == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	diarizationMgr := asr.NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		t.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 创建测试路由
	router := setupTestRouter(asrMgr, diarizationMgr)

	// 测试用例
	testCases := []struct {
		name              string
		audioFile         string
		enableDiarization bool
	}{
		{
			name:              "Basic ASR without diarization",
			audioFile:         getTestAudioPath("test.wav"),
			enableDiarization: false,
		},
		{
			name:              "ASR with diarization",
			audioFile:         getTestAudioPath("test.wav"),
			enableDiarization: true,
		},
		{
			name:              "MP4 file without diarization",
			audioFile:         getTestAudioPath("test.mp4"),
			enableDiarization: false,
		},
		{
			name:              "MP4 file with diarization",
			audioFile:         getTestAudioPath("test.mp4"),
			enableDiarization: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 加载测试音频
			audioData := loadTestAudio(t, tc.audioFile)
			if audioData == "" {
				return // 跳过测试
			}

			// 构造请求
			requestBody := OfflineASRRequest{
				Audio:             audioData,
				SampleRate:        16000,
				EnableDiarization: tc.enableDiarization,
			}

			jsonData, err := json.Marshal(requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// 创建请求
			req, err := http.NewRequest("POST", "/api/v1/asr/offline", bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 检查响应状态
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
				t.Logf("Response body: %s", w.Body.String())
				return
			}

			// 解析响应
			var response OfflineASRResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			// 验证响应
			if response.Error != "" {
				t.Errorf("API returned error: %s", response.Error)
				return
			}

			t.Logf("Recognition result: %s", response.Text)
			t.Logf("Duration: %.2f seconds", response.Duration)

			if tc.enableDiarization {
				if len(response.Segments) == 0 {
					t.Log("No speaker segments detected")
				} else {
					t.Logf("Found %d speaker segments:", len(response.Segments))
					for i, seg := range response.Segments {
						t.Logf("  Segment %d: Speaker %d, Time: %.2fs - %.2fs, Text: %s",
							i, seg.Speaker, seg.Start, seg.End, seg.Text)
					}
				}
			}
		})
	}
}

func TestOfflineASRAPIWithFile(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建管理器
	asrMgr := asr.NewOfflineASRManager(cfg)
	if asrMgr == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	diarizationMgr := asr.NewDiarizationManager(cfg)
	if diarizationMgr == nil {
		t.Fatal("Failed to create diarization manager")
	}
	defer diarizationMgr.Close()

	// 创建测试路由
	router := setupTestRouter(asrMgr, diarizationMgr)

	testFiles := []string{getTestAudioPath("test.wav"), getTestAudioPath("test.mp4")}

	for _, filename := range testFiles {
		t.Run("File upload: "+filename, func(t *testing.T) {
			// 检查文件是否存在
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", filename)
				return
			}

			// 读取文件
			fileData, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// 创建multipart请求
			body := &bytes.Buffer{}
			body.Write(fileData)

			req, err := http.NewRequest("POST", "/api/v1/asr/offline", body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "multipart/form-data")

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 检查响应
			t.Logf("Status: %d", w.Code)
			t.Logf("Response: %s", w.Body.String())
		})
	}
}

func TestOfflineASRAPIErrors(t *testing.T) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 创建管理器
	asrMgr := asr.NewOfflineASRManager(cfg)
	if asrMgr == nil {
		t.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	// 创建测试路由
	router := setupTestRouter(asrMgr, nil)

	testCases := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:           "Empty request",
			requestBody:    OfflineASRRequest{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid base64",
			requestBody: OfflineASRRequest{
				Audio:      "invalid-base64!@#$",
				SampleRate: 16000,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "not json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var jsonData []byte
			var err error

			if str, ok := tc.requestBody.(string); ok {
				jsonData = []byte(str)
			} else {
				jsonData, err = json.Marshal(tc.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/api/v1/asr/offline", bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
				t.Logf("Response: %s", w.Body.String())
			}
		})
	}
}

func BenchmarkOfflineASRAPI(b *testing.B) {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		b.Fatalf("Failed to load config: %v", err)
	}

	// 创建管理器
	asrMgr := asr.NewOfflineASRManager(cfg)
	if asrMgr == nil {
		b.Fatal("Failed to create offline ASR manager")
	}
	defer asrMgr.Close()

	// 创建测试路由
	router := setupTestRouter(asrMgr, nil)

	// 准备测试音频
	samples := make([]float32, 16000*2) // 2秒音频
	audioBytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		intSample := int16(sample * 32767.0)
		audioBytes[i*2] = byte(intSample & 0xff)
		audioBytes[i*2+1] = byte((intSample >> 8) & 0xff)
	}
	audioData := base64.StdEncoding.EncodeToString(audioBytes)

	requestBody := OfflineASRRequest{
		Audio:      audioData,
		SampleRate: 16000,
	}
	jsonData, _ := json.Marshal(requestBody)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/asr/offline", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Request failed with status %d", w.Code)
		}
	}
}

// Helper function to read response body
func readResponseBody(t *testing.T, body io.Reader) string {
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	return string(data)
}
