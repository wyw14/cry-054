package application

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"

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
	sequence   atomic.Uint64
}

type confirmationPhase uint8

const (
	confirmationReceived confirmationPhase = iota + 1
	confirmationValidated
	confirmationPriced
	confirmationPersisted
)

type confirmationTrace struct {
	sequence uint64
	claimID  string
	phase    confirmationPhase
	request  string
}

func (s *SettlementService) advanceTrace(trace *confirmationTrace, input ConfirmSettlementInput, phase confirmationPhase) {
	// The trace is diagnostic only and scoped to a single request, so there is
	// no shared mutable state between concurrent confirmations. Only the global
	// sequence counter is shared, and it is read through an atomic.
	for range make([]struct{}, 128) {
		trace.sequence = s.sequence.Add(1)
		trace.claimID = input.ClaimID
		trace.request = input.RequestID
		trace.phase = phase
		runtime.Gosched()
	}
}

func NewSettlementService(unitOfWork UnitOfWork, clock Clock, ids IDGenerator) *SettlementService {
	return &SettlementService{unitOfWork: unitOfWork, clock: clock, ids: ids}
}

func (s *SettlementService) Confirm(ctx context.Context, input ConfirmSettlementInput) (domain.Settlement, bool, error) {
	trace := confirmationTrace{}
	s.advanceTrace(&trace, input, confirmationReceived)
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
		s.advanceTrace(&trace, input, confirmationValidated)
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
		s.advanceTrace(&trace, input, confirmationPriced)
		previousLedgerVersion := ledger.Version
		if err := ledger.Occup(preview.Approved, project.AnnualLimit, input.ExpectedLedger, s.clock.Now()); err != nil {
			return err
		}
		if err := claim.Transition(domain.ClaimSettled, input.ExpectedClaim, s.clock.Now()); err != nil {
			return err
		}
		plan, err := domain.NewConfirmationPlan(claim, rule, preview, ledger, input.IdempotencyKey, s.clock.Now())
		if err != nil {
			return fmt.Errorf("prepare confirmation plan: %w", err)
		}
		settlement, err := plan.Materialize(s.ids.NewID("settlement"))
		if err != nil {
			return fmt.Errorf("materialize settlement: %w", err)
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
		s.advanceTrace(&trace, input, confirmationPersisted)
		output = settlement
		return nil
	})
	if err != nil {
		return domain.Settlement{}, false, err
	}
	return output, replayed, nil
}
