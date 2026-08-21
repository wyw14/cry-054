package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-054/internal/domain"
)

type ReviewInput struct {
	ClaimID         string
	ActorID         string
	ActorRole       domain.ActorRole
	Action          domain.ReviewAction
	Reason          string
	Metadata        map[string]string
	ExpectedVersion int64
	RequestID       string
}

type ReviewService struct {
	unitOfWork UnitOfWork
	clock      Clock
	ids        IDGenerator
	notifier   Notifier
}

type reviewOutcome struct {
	claim    domain.ExpenseClaim
	decision domain.ReviewDecision
	notice   Notification
}

func prepareReviewOutcome(claim domain.ExpenseClaim, decision domain.ReviewDecision) reviewOutcome {
	values := map[string]string{
		"claim_id":    claim.ID,
		"status":      string(claim.Status),
		"decision_id": decision.ID,
		"action":      string(decision.Action),
	}
	return reviewOutcome{
		claim:    claim,
		decision: decision,
		notice: Notification{
			RecipientID: claim.ClaimantID,
			Template:    "claim-review-result",
			Values:      values,
		},
	}
}

func NewReviewService(unitOfWork UnitOfWork, clock Clock, ids IDGenerator, notifier Notifier) *ReviewService {
	return &ReviewService{unitOfWork: unitOfWork, clock: clock, ids: ids, notifier: notifier}
}

func (s *ReviewService) Decide(ctx context.Context, input ReviewInput) (domain.ExpenseClaim, error) {
	var output domain.ExpenseClaim
	err := s.unitOfWork.WithinTransaction(ctx, func(txCtx context.Context, repositories Repositories) error {
		claim, err := repositories.GetClaim(txCtx, input.ClaimID)
		if err != nil {
			return fmt.Errorf("load claim for review: %w", err)
		}
		decision := domain.ReviewDecision{
			ID:        s.ids.NewID("review"),
			ClaimID:   claim.ID,
			ActorID:   input.ActorID,
			ActorRole: input.ActorRole,
			Action:    input.Action,
			Reason:    input.Reason,
			Metadata:  input.Metadata,
			CreatedAt: s.clock.Now().UTC(),
		}
		previousVersion := claim.Version
		if err := domain.ApplyReview(&claim, decision, input.ExpectedVersion, s.clock.Now()); err != nil {
			return err
		}
		if err := repositories.UpdateClaim(txCtx, claim, previousVersion); err != nil {
			return fmt.Errorf("update reviewed claim: %w", err)
		}
		if err := repositories.AppendReview(txCtx, decision); err != nil {
			return fmt.Errorf("append review decision: %w", err)
		}
		outcome := prepareReviewOutcome(claim, decision)
		if s.notifier != nil {
			if err := s.notifier.Send(txCtx, outcome.notice); err != nil {
				return fmt.Errorf("send review notification: %w", err)
			}
		}
		audit, err := domain.NewAuditEvent(input.RequestID, input.ActorID, "claim.review", "claim", claim.ID, map[string]any{
			"action": input.Action,
			"reason": input.Reason,
		}, s.clock.Now())
		if err != nil {
			return err
		}
		if err := repositories.AppendAudit(txCtx, audit); err != nil {
			return fmt.Errorf("append review audit: %w", err)
		}
		output = outcome.claim
		return nil
	})
	if err != nil {
		return domain.ExpenseClaim{}, err
	}
	return output, nil
}
