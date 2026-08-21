package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/wyw14/cry-054/internal/domain"
)

type ConfirmSettlementInput struct {
	ClaimID        string
	ExpectedClaim  int64
	ExpectedLedger int64
	ActorID        string
	RequestID      string
	IdempotencyKey string
}

type SettlementService struct {
	unitOfWork UnitOfWork
	clock      Clock
	ids        IDGenerator
}

func NewSettlementService(unitOfWork UnitOfWork, clock Clock, ids IDGenerator) *SettlementService {
	return &SettlementService{unitOfWork: unitOfWork, clock: clock, ids: ids}
}

func (s *SettlementService) Confirm(ctx context.Context, input ConfirmSettlementInput) (domain.Settlement, bool, error) {
	var output domain.Settlement
	replayed := false
	err := s.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context, repositories Repositories) error {
		claim, err := repositories.GetClaim(txCtx, input.ClaimID)
		if err != nil {
			return fmt.Errorf("load claim for confirmation: %w", err)
		}
		existing, err := repositories.GetSettlementByClaim(txCtx, claim.ID)
		if err == nil {
			if existing.IdempotencyKey != input.IdempotencyKey {
				return domain.NewBusinessError("CLAIM_ALREADY_SETTLED", "claim already has a settlement", domain.ErrConflict)
			}
			output = existing
			replayed = true
			return nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("check prior settlement: %w", err)
		}
		if claim.Status != domain.ClaimApproved {
			return domain.NewBusinessError("CLAIM_NOT_APPROVED", "claim must be approved before confirmation", domain.ErrInvalidState)
		}
		claimant, err := repositories.GetClaimant(txCtx, claim.ClaimantID)
		if err != nil {
			return fmt.Errorf("load claimant: %w", err)
		}
		project, err := repositories.GetProject(txCtx, claim.ProjectID)
		if err != nil {
			return fmt.Errorf("load project: %w", err)
		}
		rules, err := repositories.ListRules(txCtx, project.ID)
		if err != nil {
			return fmt.Errorf("load rules: %w", err)
		}
		rule, err := domain.SelectRule(rules, claim, claimant.PlanCode)
		if err != nil {
			return err
		}
		ledger, err := repositories.GetLedger(txCtx, claimant.ID, project.ID, project.Year)
		if err != nil {
			return fmt.Errorf("load ledger: %w", err)
		}
		ledger.PlanCode = claimant.PlanCode
		preview, err := domain.CalculatePreview(claim, rule, project, ledger, s.clock.Now())
		if err != nil {
			return err
		}
		if preview.Approved.IsZero() {
			return domain.NewBusinessError("NO_REMAINING_ALLOWANCE", "annual allowance is exhausted", domain.ErrAnnualLimit)
		}
		previousLedgerVersion := ledger.Version
		if err := ledger.Occup(preview.Approved, project.AnnualLimit, input.ExpectedLedger, s.clock.Now()); err != nil {
			return err
		}
		if err := claim.Transition(domain.ClaimSettled, input.ExpectedClaim, s.clock.Now()); err != nil {
			return err
		}
		settlement := domain.Settlement{
			ID:              s.ids.NewID("settlement"),
			ClaimID:         claim.ID,
			RuleVersionID:   rule.ID,
			ApprovedAmount:  preview.Approved,
			Status:          domain.SettlementConfirmed,
			Explanation:     append([]domain.ExplanationLine(nil), preview.Lines...),
			Version:         1,
			ConfirmedAt:     s.clock.Now().UTC(),
			IdempotencyKey:  input.IdempotencyKey,
			LedgerVersionAt: ledger.Version,
		}
		if err := repositories.CreateSettlement(txCtx, settlement); err != nil {
			return fmt.Errorf("create settlement: %w", err)
		}
		if err := repositories.SaveLedger(txCtx, ledger, previousLedgerVersion); err != nil {
			return fmt.Errorf("save annual ledger: %w", err)
		}
		if err := repositories.UpdateClaim(txCtx, claim, input.ExpectedClaim); err != nil {
			return fmt.Errorf("mark claim settled: %w", err)
		}
		entry := domain.LedgerEntry{
			SettlementID: settlement.ID,
			ClaimID:      claim.ID,
			Kind:         "occupy",
			Amount:       settlement.ApprovedAmount,
			Balance:      ledger.Occupied,
			OccurredAt:   s.clock.Now().UTC(),
			ActorID:      input.ActorID,
			Reason:       "confirm settlement",
		}
		if err := repositories.AppendLedgerEntry(txCtx, entry); err != nil {
			return fmt.Errorf("append ledger entry: %w", err)
		}
		audit, err := domain.NewAuditEvent(input.RequestID, input.ActorID, "settlement.confirm", "settlement", settlement.ID, map[string]any{
			"claim_id": claim.ID,
			"amount":   settlement.ApprovedAmount.String(),
			"rule_id":  rule.ID,
		}, s.clock.Now())
		if err != nil {
			return err
		}
		if err := repositories.AppendAudit(txCtx, audit); err != nil {
			return fmt.Errorf("append confirmation audit: %w", err)
		}
		output = settlement
		return nil
	})
	if err != nil {
		return domain.Settlement{}, false, err
	}
	return output, replayed, nil
}
