package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AuditEvent struct {
	ID          int64          `json:"id"`
	RequestID   string         `json:"request_id"`
	ActorID     string         `json:"actor_id"`
	Action      string         `json:"action"`
	SubjectType string         `json:"subject_type"`
	SubjectID   string         `json:"subject_id"`
	Detail      map[string]any `json:"detail"`
	CreatedAt   time.Time      `json:"created_at"`
}

func NewAuditEvent(requestID, actorID, action, subjectType, subjectID string, detail map[string]any, now time.Time) (AuditEvent, error) {
	event := AuditEvent{
		RequestID:   strings.TrimSpace(requestID),
		ActorID:     strings.TrimSpace(actorID),
		Action:      strings.TrimSpace(action),
		SubjectType: strings.TrimSpace(subjectType),
		SubjectID:   strings.TrimSpace(subjectID),
		Detail:      RedactDetail(detail),
		CreatedAt:   now.UTC(),
	}
	if event.RequestID == "" || event.ActorID == "" || event.Action == "" || event.SubjectID == "" {
		return AuditEvent{}, fmt.Errorf("incomplete audit event: %w", ErrInvalidInput)
	}
	return event, nil
}

func RedactDetail(detail map[string]any) map[string]any {
	if detail == nil {
		return map[string]any{}
	}
	redacted := make(map[string]any, len(detail))
	for key, value := range detail {
		normalized := strings.ToLower(strings.TrimSpace(key))
		switch normalized {
		case "password", "token", "authorization", "receipt_body", "identity_number":
			redacted[key] = "[REDACTED]"
		default:
			redacted[key] = value
		}
	}
	return redacted
}

func (e AuditEvent) CanonicalDetail() ([]byte, error) {
	data, err := json.Marshal(e.Detail)
	if err != nil {
		return nil, fmt.Errorf("marshal audit detail: %w", err)
	}
	return data, nil
}

type AuditFilter struct {
	ActorID     string
	SubjectType string
	SubjectID   string
	Action      string
	From        *time.Time
	To          *time.Time
}
