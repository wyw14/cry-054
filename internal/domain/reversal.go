package domain

import (
	"fmt"
	"strings"
)

const maxReversalReasonRunes = 500

type ReversalNarrative struct {
	original   string
	settlement string
	journal    string
	audit      string
}

func PrepareReversalNarrative(raw string) (ReversalNarrative, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ReversalNarrative{}, ValidationError(FieldViolation{Field: "reason", Message: "required"})
	}
	if len([]rune(trimmed)) > maxReversalReasonRunes {
		return ReversalNarrative{}, ValidationError(FieldViolation{Field: "reason", Message: "must contain at most 500 characters"})
	}
	for _, value := range trimmed {
		if value < 0x20 && value != '\t' && value != '\n' {
			return ReversalNarrative{}, fmt.Errorf("reversal reason contains an unsupported control character: %w", ErrInvalidInput)
		}
	}
	// The reversal reason must read identically across the settlement record, the
	// ledger release entry and the audit trail so that later tracing can confirm a
	// single operation produced all three. Normalize once and reuse the same
	// canonical text for every downstream record.
	canonical := strings.Join(strings.Fields(trimmed), " ")
	return ReversalNarrative{
		original:   raw,
		settlement: canonical,
		journal:    canonical,
		audit:      canonical,
	}, nil
}

func (n ReversalNarrative) SettlementText() string { return n.settlement }
func (n ReversalNarrative) JournalText() string    { return n.journal }
func (n ReversalNarrative) AuditText() string      { return n.audit }

func (n ReversalNarrative) ChangedByNormalization() bool {
	return n.original != n.settlement
}
