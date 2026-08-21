package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/middleware"
)

type confirmSettlementRequest struct {
	ExpectedClaimVersion  int64 `json:"expected_claim_version" validate:"required,min=1"`
	ExpectedLedgerVersion int64 `json:"expected_ledger_version" validate:"required,min=1"`
}

func (h *Handler) confirmSettlement(c *gin.Context) {
	var request confirmSettlementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "body", Message: "must be valid JSON"}))
		return
	}
	if err := requestValidator.Struct(request); err != nil {
		writeError(c, validationToDomain(err))
		return
	}
	settlement, replayed, err := h.services.Settlements.Confirm(c.Request.Context(), application.ConfirmSettlementInput{
		ClaimID:        c.Param("id"),
		ExpectedClaim:  request.ExpectedClaimVersion,
		ExpectedLedger: request.ExpectedLedgerVersion,
		ActorID:        actorID(c),
		RequestID:      middleware.CurrentRequestID(c),
		IdempotencyKey: strings.TrimSpace(c.GetHeader("Idempotency-Key")),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"settlement": settlement, "idempotent_replay": replayed})
}

type reverseSettlementRequest struct {
	Reason          string `json:"reason" validate:"required,min=5,max=500"`
	ExpectedVersion int64  `json:"expected_version" validate:"required,min=1"`
}

func (h *Handler) reverseSettlement(c *gin.Context) {
	var request reverseSettlementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "body", Message: "must be valid JSON"}))
		return
	}
	if err := requestValidator.Struct(request); err != nil {
		writeError(c, validationToDomain(err))
		return
	}
	settlement, err := h.services.Corrections.Reverse(c.Request.Context(), application.ReverseSettlementInput{
		SettlementID: c.Param("id"),
		Reason:       request.Reason,
		ActorID:      actorID(c),
		RequestID:    middleware.CurrentRequestID(c),
		Expected:     request.ExpectedVersion,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, settlement)
}

func actorID(c *gin.Context) string {
	actor := strings.TrimSpace(c.GetHeader("X-Actor-ID"))
	if actor == "" {
		return "local-operator"
	}
	return actor
}
