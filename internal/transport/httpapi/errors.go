package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/middleware"
)

type errorResponse struct {
	Code        string                  `json:"code"`
	Message     string                  `json:"message"`
	FieldErrors []domain.FieldViolation `json:"field_errors,omitempty"`
	RequestID   string                  `json:"request_id"`
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidState), errors.Is(err, domain.ErrDuplicateClaim), errors.Is(err, domain.ErrAnnualLimit):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrRuleNotApplicable):
		status = http.StatusUnprocessableEntity
	}
	message := "request could not be completed"
	var business *domain.BusinessError
	fieldErrors := []domain.FieldViolation(nil)
	if errors.As(err, &business) {
		message = business.Message
		fieldErrors = business.Violations
	}
	c.AbortWithStatusJSON(status, errorResponse{
		Code:        domain.ErrorCode(err),
		Message:     message,
		FieldErrors: fieldErrors,
		RequestID:   middleware.CurrentRequestID(c),
	})
}
