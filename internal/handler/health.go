package handler

import (
	"net/http"

	"airecorder/internal/version"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查端点
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "airecorder",
		"version": version.Short(),
	})
}

// Index 首页
func Index(c *gin.Context) {
	versionInfo := version.Get()
	c.JSON(http.StatusOK, gin.H{
		"service": "AI Recorder - Speech Recognition Service",
		"version": versionInfo.Version,
		"build_info": gin.H{
			"git_commit": versionInfo.GitCommit,
			"build_time": versionInfo.BuildTime,
			"go_version": versionInfo.GoVersion,
			"platform":   versionInfo.Platform,
		},
		"endpoints": gin.H{
			"streaming_asr":            "/api/v1/streaming/asr (WebSocket)",
			"offline_asr":              "/api/v1/offline/asr (POST)",
			"offline_with_diarization": "/api/v1/offline/asr/diarization (POST)",
			"diarization":              "/api/v1/diarization (POST)",
			"stats":                    "/api/v1/stats (GET)",
		},
	})
}
