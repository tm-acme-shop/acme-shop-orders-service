package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// Health handles GET /health
func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "orders-service",
	})
}

// Ready handles GET /ready
func (h *Handlers) Ready(c *gin.Context) {
	// TODO(TEAM-PLATFORM): Add actual readiness checks (DB, Redis, Kafka)
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "orders-service",
	})
}

// Live handles GET /live
func (h *Handlers) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// Metrics handles GET /metrics (Prometheus format)
func (h *Handlers) Metrics(c *gin.Context) {
	// TODO(TEAM-PLATFORM): Use prometheus client library
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(http.StatusOK, gin.H{
		"uptime_seconds":    time.Since(startTime).Seconds(),
		"goroutines":        runtime.NumGoroutine(),
		"heap_alloc_bytes":  m.HeapAlloc,
		"heap_sys_bytes":    m.HeapSys,
		"heap_objects":      m.HeapObjects,
		"gc_runs":           m.NumGC,
		"go_version":        runtime.Version(),
	})
}

// Version handles GET /version
func (h *Handlers) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    "1.0.0",
		"service":    "orders-service",
		"go_version": runtime.Version(),
		"built_at":   startTime.Format(time.RFC3339),
	})
}

// Debug handles GET /debug
func (h *Handlers) Debug(c *gin.Context) {
	// TODO(TEAM-SEC): Disable in production
	c.JSON(http.StatusOK, gin.H{
		"features": gin.H{
			"enable_v1_api":           h.config.Features.EnableV1API,
			"enable_legacy_payments":  h.config.Features.EnableLegacyPayments,
			"enable_order_events":     h.config.Features.EnableOrderEvents,
			"enable_order_caching":    h.config.Features.EnableOrderCaching,
		},
		"config": gin.H{
			"server_port":         h.config.Server.Port,
			"database_host":       h.config.Database.Host,
			"redis_host":          h.config.Redis.Host,
			"payment_service_url": h.config.PaymentService.BaseURL,
			"user_service_url":    h.config.UserService.BaseURL,
		},
	})
}
