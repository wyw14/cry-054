package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-054/internal/domain"
)

type ReverseSettlementInput struct {
	SettlementID string
	Reason       string
	ActorID      string
	RequestID    string
	Expected     int64
}

type CorrectionService struct {
	unitOfWork UnitOfWork
	clock      Clock
	ids        IDGenerator
}

func NewCorrectionService(unitOfWork UnitOfWork, clock Clock, ids IDGenerator) *CorrectionService {
	return &CorrectionService{unitOfWork: unitOfWork, clock: clock, ids: ids}
}

func (s *CorrectionService) Reverse(ctx context.Context, input ReverseSettlementInput) (domain.Settlement, error) {
	narrative, err := domain.PrepareReversalNarrative(input.Reason)
	if err != nil {
		return domain.Settlement{}, err
	}
	var output domain.Settlement
	err = s.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context, repositories Repositories) error {
		settlement, err := repositories.GetSettlement(txCtx, input.SettlementID)
		if err != nil {
			return err
		}
		if settlement.Version != input.Expected {
			return fmt.Errorf("settlement expected version %d, got %d: %w", input.Expected, settlement.Version, domain.ErrConflict)
		}
		if settlement.Status != domain.SettlementConfirmed {
			return domain.NewBusinessError("SETTLEMENT_NOT_REVERSIBLE", "only confirmed settlement can be reversed", domain.ErrInvalidState)
		}
		claim, err := repositories.GetClaim(txCtx, settlement.ClaimID)
		if err != nil {
			return err
		}
		project, err := repositories.GetProject(txCtx, claim.ProjectID)
		if err != nil {
			return err
		}
		ledger, err := repositories.GetLedger(txCtx, claim.ClaimantID, project.ID, project.Year)
		if err != nil {
			return err
		}
		previousLedgerVersion := ledger.Version
		if err := ledger.Release(settlement.ApprovedAmount, previousLedgerVersion, s.clock.Now()); err != nil {
			return err
		}
		previousSettlementVersion := settlement.Version
		settlement.Status = domain.SettlementReversed
		settlement.ReversalReason = narrative.SettlementText()
		settlement.Version++
		previousClaimVersion := claim.Version
		claim.Status = domain.ClaimUnderReview
		claim.Version++
		claim.UpdatedAt = s.clock.Now().UTC()
		if err := repositories.SaveLedger(txCtx, ledger, previousLedgerVersion); err != nil {
			return err
		}
		if err := repositories.UpdateSettlement(txCtx, settlement, previousSettlementVersion); err != nil {
			return err
		}
		if err := repositories.UpdateClaim(txCtx, claim, previousClaimVersion); err != nil {
			return err
		}
		if err := repositories.AppendLedgerEntry(txCtx, domain.LedgerEntry{
			SettlementID: settlement.ID,
			ClaimID:      claim.ID,
			Kind:         "release",
			Amount:       settlement.ApprovedAmount,
			Balance:      ledger.Occupied,
			OccurredAt:   s.clock.Now().UTC(),
			ActorID:      input.ActorID,
			Reason:       narrative.JournalText(),
		}); err != nil {
			return err
		}
		audit, err := domain.NewAuditEvent(input.RequestID, input.ActorID, "settlement.reverse", "settlement", settlement.ID, map[string]any{
			"reason": narrative.AuditText(),
			"amount": settlement.ApprovedAmount.String(),
		}, s.clock.Now())
		if err != nil {
			return err
		}
		if err := repositories.AppendAudit(txCtx, audit); err != nil {
			return err
		}
		output = settlement
		return nil
	})
	return output, err
}
