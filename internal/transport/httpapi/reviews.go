package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/middleware"
)

type reviewRequest struct {
	Role            domain.ActorRole    `json:"role" validate:"required"`
	Action          domain.ReviewAction `json:"action" validate:"required"`
	Reason          string              `json:"reason" validate:"required,min=3,max=500"`
	Metadata        map[string]string   `json:"metadata"`
	ExpectedVersion int64               `json:"expected_version" validate:"required,min=1"`
}

func (h *Handler) reviewClaim(c *gin.Context) {
	var request reviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "body", Message: "must be valid JSON"}))
		return
	}
	if err := requestValidator.Struct(request); err != nil {
		writeError(c, validationToDomain(err))
		return
	}
	claim, err := h.services.Reviews.Decide(c.Request.Context(), application.ReviewInput{
		ClaimID:         c.Param("id"),
		ActorID:         actorID(c),
		ActorRole:       request.Role,
		Action:          request.Action,
		Reason:          request.Reason,
		Metadata:        request.Metadata,
		ExpectedVersion: request.ExpectedVersion,
		RequestID:       middleware.CurrentRequestID(c),
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, claim)
}

func (h *Handler) reconcileLedger(c *gin.Context) {
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil || year < 2000 || year > 2200 {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "year", Message: "must be a supported year"}))
		return
	}
	report, err := h.services.Reconciliation.Build(c.Request.Context(), c.Param("claimant"), c.Param("project"), year)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, report)
}
