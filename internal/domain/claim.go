package domain

import (
	"fmt"
	"strings"
	"time"
)

type ClaimStatus string

const (
	ClaimDraft          ClaimStatus = "draft"
	ClaimSubmitted      ClaimStatus = "submitted"
	ClaimUnderReview    ClaimStatus = "under_review"
	ClaimNeedSupplement ClaimStatus = "need_supplement"
	ClaimApproved       ClaimStatus = "approved"
	ClaimRejected       ClaimStatus = "rejected"
	ClaimSettled        ClaimStatus = "settled"
	ClaimCancelled      ClaimStatus = "cancelled"
)

type ExpenseClaim struct {
	ID             string      `json:"id"`
	ClaimantID     string      `json:"claimant_id"`
	ProjectID      string      `json:"project_id"`
	Category       string      `json:"category"`
	ReceiptSummary string      `json:"receipt_summary"`
	ReceiptDigest  string      `json:"receipt_digest"`
	OccurredOn     time.Time   `json:"occurred_on"`
	Amount         Money       `json:"amount"`
	Status         ClaimStatus `json:"status"`
	Version        int64       `json:"version"`
	IdempotencyKey string      `json:"idempotency_key"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func NewExpenseClaim(
	id, claimantID, projectID, category, receiptSummary, receiptDigest string,
	occurredOn time.Time,
	amount Money,
	idempotencyKey string,
	now time.Time,
) (ExpenseClaim, error) {
	claim := ExpenseClaim{
		ID:             strings.TrimSpace(id),
		ClaimantID:     strings.TrimSpace(claimantID),
		ProjectID:      strings.TrimSpace(projectID),
		Category:       strings.ToLower(strings.TrimSpace(category)),
		ReceiptSummary: strings.TrimSpace(receiptSummary),
		ReceiptDigest:  strings.TrimSpace(receiptDigest),
		OccurredOn:     dateOnly(occurredOn),
		Amount:         amount,
		Status:         ClaimSubmitted,
		Version:        1,
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}
	if claim.ID == "" || claim.ClaimantID == "" || claim.ProjectID == "" {
		return ExpenseClaim{}, fmt.Errorf("claim identity: %w", ErrInvalidInput)
	}
	if claim.Category == "" || claim.ReceiptDigest == "" || claim.IdempotencyKey == "" {
		return ExpenseClaim{}, fmt.Errorf("claim evidence: %w", ErrInvalidInput)
	}
	if !claim.Amount.Positive() {
		return ExpenseClaim{}, fmt.Errorf("claim amount: %w", ErrInvalidInput)
	}
	return claim, nil
}

func (c *ExpenseClaim) Transition(next ClaimStatus, expectedVersion int64, now time.Time) error {
	if c.Version != expectedVersion {
		return fmt.Errorf("claim %s expected version %d, got %d: %w", c.ID, expectedVersion, c.Version, ErrConflict)
	}
	if !allowedClaimTransition(c.Status, next) {
		return fmt.Errorf("claim %s cannot move from %s to %s: %w", c.ID, c.Status, next, ErrInvalidState)
	}
	c.Status = next
	c.Version++
	c.UpdatedAt = now.UTC()
	return nil
}

func allowedClaimTransition(from, to ClaimStatus) bool {
	allowed := map[ClaimStatus]map[ClaimStatus]bool{
		ClaimDraft:          {ClaimSubmitted: true, ClaimCancelled: true},
		ClaimSubmitted:      {ClaimUnderReview: true, ClaimCancelled: true},
		ClaimUnderReview:    {ClaimNeedSupplement: true, ClaimApproved: true, ClaimRejected: true},
		ClaimNeedSupplement: {ClaimSubmitted: true, ClaimCancelled: true},
		ClaimApproved:       {ClaimSettled: true, ClaimUnderReview: true},
		ClaimRejected:       {ClaimUnderReview: true},
		ClaimSettled:        {},
		ClaimCancelled:      {},
	}
	return allowed[from][to]
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
