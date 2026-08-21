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

func NewPreviewService(repositories Repositories, clock Clock) *PreviewService {
	return &PreviewService{repositories: repositories, clock: clock}
}

func (s *PreviewService) Preview(ctx context.Context, claimID string) (domain.Preview, error) {
	claim, err := s.repositories.GetClaim(ctx, claimID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("load claim: %w", err)
	}
	if claim.Status != domain.ClaimApproved && claim.Status != domain.ClaimUnderReview && claim.Status != domain.ClaimSubmitted {
		return domain.Preview{}, domain.NewBusinessError("CLAIM_NOT_PREVIEWABLE", "claim cannot be previewed in current state", domain.ErrInvalidState)
	}
	claimant, err := s.repositories.GetClaimant(ctx, claim.ClaimantID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("load claimant: %w", err)
	}
	project, err := s.repositories.GetProject(ctx, claim.ProjectID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("load project: %w", err)
	}
	rules, err := s.repositories.ListRules(ctx, claim.ProjectID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("list rules: %w", err)
	}
	rule, err := domain.SelectRule(rules, claim, claimant.PlanCode)
	if err != nil {
		return domain.Preview{}, err
	}
	ledger, err := s.repositories.GetLedger(ctx, claimant.ID, project.ID, project.Year)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("load ledger: %w", err)
	}
	ledger.PlanCode = claimant.PlanCode
	preview, err := domain.CalculatePreview(claim, rule, project, ledger, s.clock.Now())
	if err != nil {
		return domain.Preview{}, fmt.Errorf("calculate preview: %w", err)
	}
	return preview, nil
}
