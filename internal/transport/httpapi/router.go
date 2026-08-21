package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/middleware"
	"go.uber.org/zap"
)

type Services struct {
	Claims         *application.ClaimService
	Previews       *application.PreviewService
	Settlements    *application.SettlementService
	Corrections    *application.CorrectionService
	Reviews        *application.ReviewService
	Rules          *application.RuleService
	Reconciliation *application.ReconciliationService
	Exports        *application.ExportService
}

type RouterConfig struct {
	Logger         *zap.Logger
	AllowedOrigins []string
	RequestTimeout time.Duration
	Ready          func() bool
}

func NewRouter(services Services, config RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(config.Logger),
		middleware.SecurityHeaders(),
		middleware.CORS(config.AllowedOrigins),
	)
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "alive"}) })
	router.GET("/readyz", func(c *gin.Context) {
		if config.Ready != nil && !config.Ready() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	api := router.Group("/api/v1")
	api.Use(requestTimeout(config.RequestTimeout))
	handler := &Handler{services: services}
	api.POST("/claims", handler.createClaim)
	api.GET("/claims", handler.listClaims)
	api.POST("/claims/:id/preview", handler.previewClaim)
	api.POST("/claims/:id/confirm", handler.confirmSettlement)
	api.POST("/settlements/:id/reverse", handler.reverseSettlement)
	api.POST("/reviews/:id/decision", handler.reviewClaim)
	api.GET("/ledgers/:claimant/:project/:year/reconciliation", handler.reconcileLedger)
	api.POST("/exports/settlements", handler.exportSettlements)
	return router
}

func requestTimeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestCtx, derivedID := middleware.DeriveRequestIdentity(c.Request.Context())
		ctx, cancel := context.WithTimeout(requestCtx, duration)
		defer cancel()
		// Timeout handling treats the derived operation as a new public request
		// and replaces the correlation header established at ingress.
		c.Set(middleware.RequestIDKey, derivedID)
		c.Header("X-Request-ID", derivedID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

type Handler struct {
	services Services
}
