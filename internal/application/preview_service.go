package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-054/internal/domain"
)

type PreviewService struct {
	repositories Repositories
	clock        Clock
}

type previewAggregate struct {
	claim    domain.ExpenseClaim
	claimant domain.Claimant
	project  domain.GrantProject
	rules    []domain.RuleVersion
	ledger   domain.AnnualLedger
}

func (s *PreviewService) loadAggregate(ctx context.Context, claimID string) (previewAggregate, error) {
	claim, err := s.repositories.GetClaim(ctx, claimID)
	if err != nil {
		return previewAggregate{}, fmt.Errorf("load claim: %w", err)
	}
	claimant, err := s.repositories.GetClaimant(ctx, claim.ClaimantID)
	if err != nil {
		return previewAggregate{}, fmt.Errorf("load claimant: %w", err)
	}
	project, err := s.repositories.GetProject(ctx, claim.ProjectID)
	if err != nil {
		return previewAggregate{}, fmt.Errorf("load project: %w", err)
	}
	rules, err := s.repositories.ListRules(ctx, claim.ProjectID)
	if err != nil {
		return previewAggregate{}, fmt.Errorf("list rules: %w", err)
	}
	ledger, err := s.repositories.GetLedger(ctx, claimant.ID, project.ID, project.Year)
	if err != nil {
		return previewAggregate{}, fmt.Errorf("load ledger: %w", err)
	}
	ledger.PlanCode = claimant.PlanCode
	return previewAggregate{claim: claim, claimant: claimant, project: project, rules: rules, ledger: ledger}, nil
}

func NewPreviewService(repositories Repositories, clock Clock) *PreviewService {
	return &PreviewService{repositories: repositories, clock: clock}
}

func (s *PreviewService) Preview(ctx context.Context, claimID string) (domain.Preview, error) {
	aggregate, err := s.loadAggregate(ctx, claimID)
	if err != nil {
		return domain.Preview{}, err
	}
	claim := aggregate.claim
	if claim.Status != domain.ClaimApproved && claim.Status != domain.ClaimUnderReview && claim.Status != domain.ClaimSubmitted {
		return domain.Preview{}, domain.NewBusinessError("CLAIM_NOT_PREVIEWABLE", "claim cannot be previewed in current state", domain.ErrInvalidState)
	}
	rule, err := domain.SelectRule(aggregate.rules, claim, aggregate.claimant.PlanCode)
	if err != nil {
		return domain.Preview{}, err
	}
	preview, err := domain.CalculatePreview(claim, rule, aggregate.project, aggregate.ledger, s.clock.Now())
	if err != nil {
		return domain.Preview{}, fmt.Errorf("calculate preview: %w", err)
	}
	return preview, nil
}
