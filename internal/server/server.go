package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"airecorder/internal/asr"
	"airecorder/internal/config"
	"airecorder/internal/handler"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config         *config.Config
	router         *gin.Engine
	streamingASR   *asr.StreamingASRManager
	offlineASR     *asr.OfflineASRManager
	diarizationMgr *asr.DiarizationManager
	httpServer     *http.Server
	shutdown       chan struct{}
	wg             sync.WaitGroup
}

func NewServer(cfg *config.Config) *Server {
	// 设置 Gin 模式
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// 初始化服务器
	srv := &Server{
		config:   cfg,
		router:   router,
		shutdown: make(chan struct{}),
	}

	// 初始化 ASR 管理器
	if cfg.StreamingASR.Enabled {
		srv.streamingASR = asr.NewStreamingASRManager(cfg)
	}

	if cfg.OfflineASR.Enabled {
		srv.offlineASR = asr.NewOfflineASRManager(cfg)
	}

	if cfg.SpeakerDiarization.Enabled {
		srv.diarizationMgr = asr.NewDiarizationManager(cfg)
	}

	// 设置路由
	srv.setupRoutes()

	return srv
}

func (s *Server) setupRoutes() {
	// 创建 /realkws 路由组
	realkws := s.router.Group("/realkws")
	{
		// 静态文件服务（测试页面）
		realkws.Static("/static", "./static")
		realkws.GET("/test", func(c *gin.Context) {
			c.File("./static/index.html")
		})

		// 健康检查
		realkws.GET("/health", handler.HealthCheck)
		realkws.GET("/", handler.Index)

		// API 路由组
		api := realkws.Group("/api/v1")
		{
			// 实时语音识别 WebSocket
			if s.config.StreamingASR.Enabled {
				api.GET("/streaming/asr", func(c *gin.Context) {
					handler.HandleStreamingASR(c, s.streamingASR)
				})
			}

			// 离线语音识别
			if s.config.OfflineASR.Enabled {
				// 普通模式（不带说话者分离）
				api.POST("/offline/asr", func(c *gin.Context) {
					handler.HandleOfflineASR(c, s.offlineASR, nil)
				})

				// 带说话者分离模式
				if s.config.SpeakerDiarization.Enabled {
					api.POST("/offline/asr/diarization", func(c *gin.Context) {
						handler.HandleOfflineASR(c, s.offlineASR, s.diarizationMgr)
					})
				}
			}

			// 说话者分离独立接口
			if s.config.SpeakerDiarization.Enabled {
				api.POST("/diarization", func(c *gin.Context) {
					handler.HandleDiarization(c, s.diarizationMgr)
				})
			}

			// 统计信息
			api.GET("/stats", func(c *gin.Context) {
				handler.HandleStats(c, s.streamingASR, s.offlineASR)
			})
		}
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	// 启动 HTTP 服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		log.Printf("Server listening on %s", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 等待关闭信号
	<-s.shutdown

	return s.Stop()
}

func (s *Server) Stop() error {
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// 关闭 ASR 管理器
	if s.streamingASR != nil {
		s.streamingASR.Close()
	}

	if s.offlineASR != nil {
		s.offlineASR.Close()
	}

	if s.diarizationMgr != nil {
		s.diarizationMgr.Close()
	}

	s.wg.Wait()
	log.Println("Server stopped")

	return nil
}
