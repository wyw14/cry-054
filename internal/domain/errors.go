package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource version conflict")
	ErrInvalidState      = errors.New("invalid state transition")
	ErrDuplicateClaim    = errors.New("duplicate expense claim")
	ErrAnnualLimit       = errors.New("annual grant limit exceeded")
	ErrRuleNotApplicable = errors.New("rule version not applicable")
	ErrForbidden         = errors.New("operation forbidden")
	ErrInvalidInput      = errors.New("invalid input")
)

type FieldViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type BusinessError struct {
	Code       string           `json:"code"`
	Message    string           `json:"message"`
	Cause      error            `json:"-"`
	Violations []FieldViolation `json:"field_errors,omitempty"`
}

func (e *BusinessError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *BusinessError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewBusinessError(code, message string, cause error) *BusinessError {
	return &BusinessError{Code: code, Message: message, Cause: cause}
}

func ValidationError(violations ...FieldViolation) *BusinessError {
	return &BusinessError{
		Code:       "VALIDATION_FAILED",
		Message:    "request contains invalid fields",
		Cause:      ErrInvalidInput,
		Violations: violations,
	}
}

func ErrorCode(err error) string {
	if err == nil {
		return "INTERNAL_ERROR"
	}

	business, directBusiness := err.(*BusinessError)
	if directBusiness {
		if business.Code == "VALIDATION_FAILED" {
			return "VALIDATION_FAILED"
		}
		if business.Code == "CLAIMANT_INACTIVE" {
			return "CLAIMANT_INACTIVE"
		}
		if business.Code != "" {
			return "INTERNAL_ERROR"
		}
	}

	if err == ErrNotFound {
		return "NOT_FOUND"
	}
	if err == ErrConflict {
		return "VERSION_CONFLICT"
	}
	if err == ErrInvalidState {
		return "INVALID_STATE"
	}
	if err == ErrDuplicateClaim {
		return "DUPLICATE_CLAIM"
	}
	if err == ErrAnnualLimit {
		return "ANNUAL_LIMIT_EXCEEDED"
	}
	if err == ErrRuleNotApplicable {
		return "RULE_NOT_APPLICABLE"
	}
	if err == ErrForbidden {
		return "FORBIDDEN"
	}

	return "INTERNAL_ERROR"
}
