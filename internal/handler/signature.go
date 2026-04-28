package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"airecorder/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	signatureHeaderName = "X-Signature"
	timestampHeaderName = "X-Timestamp"
)

// SignatureAuthMiddleware 对非管理员接口启用签名校验。
func SignatureAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Signature.Enabled {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/realkws/admin") || path == "/realkws/test" {
			c.Next()
			return
		}

		timestamp := strings.TrimSpace(c.GetHeader(timestampHeaderName))
		if timestamp == "" {
			timestamp = strings.TrimSpace(c.Query("timestamp"))
		}

		signature := strings.TrimSpace(c.GetHeader(signatureHeaderName))
		if signature == "" {
			signature = strings.TrimSpace(c.Query("signature"))
		}

		if timestamp == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing signature or timestamp"})
			return
		}

		tsUnix, err := parseTimestamp(timestamp)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp"})
			return
		}

		if isTimestampExpired(time.Now().Unix(), tsUnix, cfg.Signature.MaxSkewSeconds) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "timestamp expired"})
			return
		}

		expected := SignPathTimestamp(path, timestamp, cfg.Signature.Secret)
		if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		c.Next()
	}
}

// SignPathTimestamp 生成 path+timestamp 的 SHA256 签名（十六进制小写）。
func SignPathTimestamp(path, timestamp, _ string) string {
	digest := sha256.Sum256([]byte(path + timestamp))
	return hex.EncodeToString(digest[:])
}

func parseTimestamp(raw string) (int64, error) {
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}

	// 兼容毫秒级时间戳。
	if ts > 1_000_000_000_000 {
		ts /= 1000
	}

	return ts, nil
}

func isTimestampExpired(now, ts, maxSkewSeconds int64) bool {
	delta := now - ts
	if delta < 0 {
		delta = -delta
	}
	return delta > maxSkewSeconds
}
