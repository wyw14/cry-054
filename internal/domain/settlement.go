package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type ExplanationLine struct {
	From        Money           `json:"from"`
	To          Money           `json:"to"`
	Base        Money           `json:"base"`
	Rate        decimal.Decimal `json:"rate"`
	Subsidy     Money           `json:"subsidy"`
	Description string          `json:"description"`
}

type Preview struct {
	ClaimID           string            `json:"claim_id"`
	RuleVersionID     string            `json:"rule_version_id"`
	Requested         Money             `json:"requested"`
	Eligible          Money             `json:"eligible"`
	Approved          Money             `json:"approved"`
	OccupiedBefore    Money             `json:"occupied_before"`
	RemainingBefore   Money             `json:"remaining_before"`
	RemainingAfter    Money             `json:"remaining_after"`
	Lines             []ExplanationLine `json:"lines"`
	Warnings          []string          `json:"warnings,omitempty"`
	CalculatedAt      time.Time         `json:"calculated_at"`
	LedgerVersionUsed int64             `json:"ledger_version_used"`
}

func CalculatePreview(claim ExpenseClaim, rule RuleVersion, project GrantProject, ledger AnnualLedger, now time.Time) (Preview, error) {
	if claim.ProjectID != project.ID || ledger.ProjectID != project.ID || ledger.ClaimantID != claim.ClaimantID {
		return Preview{}, fmt.Errorf("preview aggregate mismatch: %w", ErrInvalidInput)
	}
	if !rule.Applies(claim, ledger.PlanCode) {
		return Preview{}, fmt.Errorf("rule %s does not apply to claim %s: %w", rule.ID, claim.ID, ErrRuleNotApplicable)
	}
	remaining := project.AnnualLimit.Sub(ledger.Occupied).Max(ZeroMoney)
	eligible := claim.Amount.Min(rule.Cap)
	lines := make([]ExplanationLine, 0, len(rule.Segments))
	approved := ZeroMoney
	consumedBase := ZeroMoney
	for index, segment := range rule.Segments {
		if eligible.LessThanOrEqual(consumedBase) {
			break
		}
		segmentEnd := eligible.Min(segment.Threshold)
		base := segmentEnd.Sub(consumedBase)
		if !base.Positive() {
			continue
		}
		subsidy := base.Mul(segment.Rate)
		approved = approved.Add(subsidy)
		lines = append(lines, ExplanationLine{
			From:        consumedBase,
			To:          segmentEnd,
			Base:        base,
			Rate:        segment.Rate,
			Subsidy:     subsidy,
			Description: fmt.Sprintf("第%d段按%s比例补助", index+1, segment.Rate.StringFixed(2)),
		})
		consumedBase = segmentEnd
	}
	warnings := make([]string, 0, 2)
	if claim.Amount.GreaterThan(rule.Cap) {
		warnings = append(warnings, "申报金额超过当前规则封顶线")
	}
	if approved.GreaterThan(remaining) {
		approved = remaining
		warnings = append(warnings, "补助金额受年度剩余额度限制")
	}
	return Preview{
		ClaimID:           claim.ID,
		RuleVersionID:     rule.ID,
		Requested:         claim.Amount,
		Eligible:          eligible,
		Approved:          approved,
		OccupiedBefore:    ledger.Occupied,
		RemainingBefore:   remaining,
		RemainingAfter:    remaining.Sub(approved).Max(ZeroMoney),
		Lines:             lines,
		Warnings:          warnings,
		CalculatedAt:      now.UTC(),
		LedgerVersionUsed: ledger.Version,
	}, nil
}

type SettlementStatus string

const (
	SettlementConfirmed SettlementStatus = "confirmed"
	SettlementReversed  SettlementStatus = "reversed"
	SettlementCorrected SettlementStatus = "corrected"
)

type Settlement struct {
	ID              string            `json:"id"`
	ClaimID         string            `json:"claim_id"`
	RuleVersionID   string            `json:"rule_version_id"`
	ApprovedAmount  Money             `json:"approved_amount"`
	Status          SettlementStatus  `json:"status"`
	Explanation     []ExplanationLine `json:"explanation"`
	Version         int64             `json:"version"`
	ConfirmedAt     time.Time         `json:"confirmed_at"`
	ReversalReason  string            `json:"reversal_reason,omitempty"`
	CorrectionOfID  string            `json:"correction_of_id,omitempty"`
	IdempotencyKey  string            `json:"idempotency_key"`
	LedgerVersionAt int64             `json:"ledger_version_at"`
}
