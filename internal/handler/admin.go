package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"airecorder/internal/asr"
	"airecorder/internal/config"

	"github.com/gin-gonic/gin"
)

const adminTokenMessage = "airecorder-admin-token"

// computeAdminToken 根据密码生成管理员 token（HMAC-SHA256）
func computeAdminToken(password string) string {
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write([]byte(adminTokenMessage))
	return hex.EncodeToString(mac.Sum(nil))
}

// AdminAuthMiddleware 验证管理员 token 的中间件
func AdminAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if token == "" {
			token = c.Query("token")
		}

		if token == "" || !hmac.Equal([]byte(token), []byte(computeAdminToken(cfg.Admin.Password))) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// HandleAdminLogin 管理员登录，验证密码并返回 token
func HandleAdminLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
			return
		}

		if req.Password != cfg.Admin.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
			return
		}

		token := computeAdminToken(cfg.Admin.Password)
		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

// HandleAdminStats 返回系统统计信息
func HandleAdminStats(streamingASR *asr.StreamingASRManager, offlineASR *asr.OfflineASRManager, taskQueue *asr.TaskQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := gin.H{}

		if streamingASR != nil {
			stats["streaming"] = streamingASR.GetStats()
		}
		if offlineASR != nil {
			stats["offline"] = offlineASR.GetStats()
		}
		if taskQueue != nil {
			stats["task_queue"] = taskQueue.GetStats()
		}

		c.JSON(http.StatusOK, stats)
	}
}

// HandleAdminListTasks 返回所有任务列表
func HandleAdminListTasks(taskQueue *asr.TaskQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		if taskQueue == nil {
			c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
			return
		}
		tasks := taskQueue.ListTasks()
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

// HandleAdminCancelTask 取消指定任务
func HandleAdminCancelTask(taskQueue *asr.TaskQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("taskId")
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "taskId required"})
			return
		}
		if !asr.IsValidTaskID(taskID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid taskId format"})
			return
		}
		if taskQueue == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "task queue not available"})
			return
		}
		if err := taskQueue.CancelTask(taskID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "task cancelled", "task_id": taskID})
	}
}

// HandleAdminListSessions 返回当前活跃的流式 ASR 会话
func HandleAdminListSessions(streamingASR *asr.StreamingASRManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if streamingASR == nil {
			c.JSON(http.StatusOK, gin.H{"sessions": []interface{}{}})
			return
		}
		sessions := streamingASR.ListSessions()
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

// HandleAdminCloseSession 强制关闭指定流式 ASR 会话
func HandleAdminCloseSession(streamingASR *asr.StreamingASRManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId required"})
			return
		}
		if streamingASR == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "streaming ASR not available"})
			return
		}
		if !streamingASR.CloseSessionByAdmin(sessionID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "session closed", "session_id": sessionID})
	}
}

// HandleAdminWorkers 返回 worker 状态
func HandleAdminWorkers(taskQueue *asr.TaskQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		if taskQueue == nil {
			c.JSON(http.StatusOK, gin.H{"workers": []interface{}{}})
			return
		}
		workers := taskQueue.GetWorkerStatus()
		c.JSON(http.StatusOK, gin.H{"workers": workers})
	}
}
