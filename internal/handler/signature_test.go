package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"airecorder/internal/config"

	"github.com/gin-gonic/gin"
)

func newSignatureTestRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SignatureAuthMiddleware(cfg))
	r.GET("/realkws/api/v1/stats", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/realkws/api/v1/streaming/asr", func(c *gin.Context) {
		c.String(http.StatusOK, "ws")
	})
	r.GET("/realkws/admin/login", func(c *gin.Context) {
		c.String(http.StatusOK, "admin")
	})
	return r
}

func TestSignatureAuthMiddlewareWithHeaders(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	path := "/realkws/api/v1/stats"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := SignPathTimestamp(path, ts, cfg.Signature.Secret)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSignatureAuthMiddlewareRejectsInvalidSignature(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	path := "/realkws/api/v1/stats"
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", "bad-signature")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSignatureAuthMiddlewareRejectsMissingSignature(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/realkws/api/v1/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSignatureAuthMiddlewareRejectsExpiredTimestamp(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	path := "/realkws/api/v1/stats"
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	sig := SignPathTimestamp(path, ts, cfg.Signature.Secret)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSignatureAuthMiddlewareAllowsAdminWithoutSignature(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/realkws/admin/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSignatureAuthMiddlewareWithQueryParams(t *testing.T) {
	cfg := &config.Config{
		Signature: config.SignatureConfig{
			Enabled:        true,
			Secret:         "test-signature-secret",
			MaxSkewSeconds: 300,
		},
	}
	r := newSignatureTestRouter(cfg)

	path := "/realkws/api/v1/streaming/asr"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := SignPathTimestamp(path, ts, cfg.Signature.Secret)

	target := path + "?timestamp=" + ts + "&signature=" + sig
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
