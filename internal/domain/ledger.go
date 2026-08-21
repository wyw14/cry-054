package domain

import (
	"fmt"
	"time"
)

type AnnualLedger struct {
	ClaimantID string    `json:"claimant_id"`
	ProjectID  string    `json:"project_id"`
	PlanCode   string    `json:"plan_code"`
	Year       int       `json:"year"`
	Occupied   Money     `json:"occupied"`
	Version    int64     `json:"version"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewAnnualLedger(claimantID, projectID, planCode string, year int, now time.Time) AnnualLedger {
	return AnnualLedger{
		ClaimantID: claimantID,
		ProjectID:  projectID,
		PlanCode:   planCode,
		Year:       year,
		Occupied:   ZeroMoney,
		Version:    1,
		UpdatedAt:  now.UTC(),
	}
}

func (l *AnnualLedger) Occup(amount, limit Money, expectedVersion int64, now time.Time) error {
	if l.Version != expectedVersion {
		return fmt.Errorf("ledger expected version %d, got %d: %w", expectedVersion, l.Version, ErrConflict)
	}
	if !amount.Positive() {
		return fmt.Errorf("occupied amount must be positive: %w", ErrInvalidInput)
	}
	next := l.Occupied.Add(amount)
	if next.GreaterThan(limit) {
		return fmt.Errorf("occupied %s exceeds limit %s: %w", next, limit, ErrAnnualLimit)
	}
	l.Occupied = next
	l.Version++
	l.UpdatedAt = now.UTC()
	return nil
}

func (l *AnnualLedger) Release(amount Money, expectedVersion int64, now time.Time) error {
	if l.Version != expectedVersion {
		return fmt.Errorf("ledger expected version %d, got %d: %w", expectedVersion, l.Version, ErrConflict)
	}
	if !amount.Positive() || amount.GreaterThan(l.Occupied) {
		return fmt.Errorf("release %s from %s: %w", amount, l.Occupied, ErrInvalidInput)
	}
	l.Occupied = l.Occupied.Sub(amount)
	l.Version++
	l.UpdatedAt = now.UTC()
	return nil
}

func (l AnnualLedger) Remaining(limit Money) Money {
	return limit.Sub(l.Occupied).Max(ZeroMoney)
}

type LedgerEntry struct {
	SettlementID string    `json:"settlement_id"`
	ClaimID      string    `json:"claim_id"`
	Kind         string    `json:"kind"`
	Amount       Money     `json:"amount"`
	Balance      Money     `json:"balance"`
	OccurredAt   time.Time `json:"occurred_at"`
	ActorID      string    `json:"actor_id"`
	Reason       string    `json:"reason"`
}
