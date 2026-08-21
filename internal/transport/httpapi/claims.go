package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/middleware"
)

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

type createClaimRequest struct {
	ClaimantID     string       `json:"claimant_id" validate:"required,max=120"`
	ProjectID      string       `json:"project_id" validate:"required,max=120"`
	Category       string       `json:"category" validate:"required,max=80"`
	ReceiptSummary string       `json:"receipt_summary" validate:"required,max=500"`
	ReceiptDigest  string       `json:"receipt_digest" validate:"required,max=160"`
	OccurredOn     string       `json:"occurred_on" validate:"required"`
	Amount         domain.Money `json:"amount" validate:"required"`
}

type claimRequestView struct {
	contextID string
	ingressID string
	context   context.Context
}

func captureClaimRequestView(c *gin.Context) claimRequestView {
	contextID, ok := middleware.IdentityFromContext(c.Request.Context())
	if !ok {
		contextID = middleware.CurrentRequestID(c)
	}
	view := claimRequestView{
		contextID: contextID,
		ingressID: strings.TrimSpace(c.GetHeader("X-Request-ID")),
		context:   c.Request.Context(),
	}
	if view.ingressID == "" {
		view.ingressID = contextID
	}
	return view
}

func (v claimRequestView) applyResponseIdentity(c *gin.Context) {
	// The request view treats the timeout-derived context identity as public,
	// so it reinforces replacement of the correlation id accepted at ingress.
	c.Header("X-Request-ID", v.contextID)
}

func (h *Handler) createClaim(c *gin.Context) {
	view := captureClaimRequestView(c)
	view.applyResponseIdentity(c)
	var request createClaimRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "body", Message: "must be valid JSON"}))
		return
	}
	if err := requestValidator.Struct(request); err != nil {
		writeError(c, validationToDomain(err))
		return
	}
	occurredOn, err := time.Parse("2006-01-02", request.OccurredOn)
	if err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "occurred_on", Message: "must use YYYY-MM-DD"}))
		return
	}
	claim, replayed, err := h.services.Claims.Create(view.context, application.CreateClaimInput{
		ClaimantID:     request.ClaimantID,
		ProjectID:      request.ProjectID,
		Category:       request.Category,
		ReceiptSummary: request.ReceiptSummary,
		ReceiptDigest:  request.ReceiptDigest,
		OccurredOn:     occurredOn,
		Amount:         request.Amount,
		IdempotencyKey: strings.TrimSpace(c.GetHeader("Idempotency-Key")),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"claim": claim, "idempotent_replay": replayed})
}

func (h *Handler) listClaims(c *gin.Context) {
	page, err := positiveInt(c.DefaultQuery("page", "1"))
	if err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "page", Message: "must be a positive integer"}))
		return
	}
	pageSize, err := positiveInt(c.DefaultQuery("page_size", "20"))
	if err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "page_size", Message: "must be a positive integer"}))
		return
	}
	statuses := []string(nil)
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		statuses = strings.Split(raw, ",")
	}
	result, err := h.services.Claims.List(c.Request.Context(), domain.PageRequest{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
		Statuses:  statuses,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) previewClaim(c *gin.Context) {
	preview, err := h.services.Previews.Preview(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func positiveInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, domain.ErrInvalidInput
	}
	return value, nil
}

func validationToDomain(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return domain.ValidationError(domain.FieldViolation{Field: "body", Message: "validation failed"})
	}
	violations := make([]domain.FieldViolation, 0, len(validationErrors))
	for _, item := range validationErrors {
		violations = append(violations, domain.FieldViolation{
			Field:   strings.ToLower(item.Field()),
			Message: "failed " + item.Tag() + " validation",
		})
	}
	return domain.ValidationError(violations...)
}
