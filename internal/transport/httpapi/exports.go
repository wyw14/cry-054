package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/middleware"
)

type exportRequest struct {
	ProjectID string           `json:"project_id" validate:"required"`
	Year      int              `json:"year" validate:"required,min=2000,max=2200"`
	Role      domain.ActorRole `json:"role" validate:"required"`
	Columns   []string         `json:"columns" validate:"max=6,dive,required"`
}

func (h *Handler) exportSettlements(c *gin.Context) {
	var request exportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ValidationError(domain.FieldViolation{Field: "body", Message: "must be valid JSON"}))
		return
	}
	if err := requestValidator.Struct(request); err != nil {
		writeError(c, validationToDomain(err))
		return
	}
	attachment, err := h.services.Exports.ExportSettlements(c.Request.Context(), application.ExportRequest{
		ProjectID: request.ProjectID,
		Year:      request.Year,
		ActorID:   actorID(c),
		ActorRole: request.Role,
		RequestID: middleware.CurrentRequestID(c),
		Columns:   request.Columns,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"export_id": attachment.ID,
		"size":      attachment.Size,
		"sha256":    attachment.SHA256,
	})
}
