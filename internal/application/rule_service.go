package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/cry-054/internal/domain"
)

type RuleService struct {
	repositories Repositories
	clock        Clock
}

func NewRuleService(repositories Repositories, clock Clock) *RuleService {
	return &RuleService{repositories: repositories, clock: clock}
}

func (s *RuleService) CreateDraft(ctx context.Context, rule domain.RuleVersion) (domain.RuleVersion, error) {
	if rule.Published() {
		return domain.RuleVersion{}, domain.NewBusinessError("RULE_ALREADY_PUBLISHED", "published rules are immutable", domain.ErrConflict)
	}
	if err := rule.Validate(); err != nil {
		return domain.RuleVersion{}, err
	}
	project, err := s.repositories.GetProject(ctx, rule.ProjectID)
	if err != nil {
		return domain.RuleVersion{}, err
	}
	if rule.EffectiveFrom.Year() != project.Year {
		return domain.RuleVersion{}, domain.ValidationError(domain.FieldViolation{Field: "effective_from", Message: "must fall in project year"})
	}
	if err := s.repositories.CreateRule(ctx, rule); err != nil {
		return domain.RuleVersion{}, fmt.Errorf("create rule version: %w", err)
	}
	return rule, nil
}

func (s *RuleService) Publish(ctx context.Context, id string) (domain.RuleVersion, error) {
	rule, err := s.repositories.GetRule(ctx, id)
	if err != nil {
		return domain.RuleVersion{}, err
	}
	published, err := rule.Publish(s.clock.Now())
	if err != nil {
		return domain.RuleVersion{}, err
	}
	if err := s.repositories.PublishRule(ctx, published); err != nil {
		return domain.RuleVersion{}, fmt.Errorf("publish rule version: %w", err)
	}
	return published, nil
}

type RuleImpact struct {
	ClaimID       string       `json:"claim_id"`
	OldApproved   domain.Money `json:"old_approved"`
	NewApproved   domain.Money `json:"new_approved"`
	Difference    domain.Money `json:"difference"`
	ChangedReason string       `json:"changed_reason"`
}

func (s *RuleService) Compare(ctx context.Context, projectID, oldRuleID, newRuleID string, claims []domain.ExpenseClaim) ([]RuleImpact, error) {
	project, err := s.repositories.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	oldRule, err := s.repositories.GetRule(ctx, oldRuleID)
	if err != nil {
		return nil, err
	}
	newRule, err := s.repositories.GetRule(ctx, newRuleID)
	if err != nil {
		return nil, err
	}
	impacts := make([]RuleImpact, 0, len(claims))
	for _, claim := range claims {
		claimant, err := s.repositories.GetClaimant(ctx, claim.ClaimantID)
		if err != nil {
			return nil, err
		}
		ledger := domain.NewAnnualLedger(claim.ClaimantID, project.ID, claimant.PlanCode, project.Year, time.Unix(0, 0))
		oldPreview, oldErr := domain.CalculatePreview(claim, oldRule, project, ledger, s.clock.Now())
		newPreview, newErr := domain.CalculatePreview(claim, newRule, project, ledger, s.clock.Now())
		if oldErr != nil && newErr != nil {
			continue
		}
		oldApproved := domain.ZeroMoney
		newApproved := domain.ZeroMoney
		if oldErr == nil {
			oldApproved = oldPreview.Approved
		}
		if newErr == nil {
			newApproved = newPreview.Approved
		}
		if oldApproved.Equal(newApproved) {
			continue
		}
		impacts = append(impacts, RuleImpact{
			ClaimID:       claim.ID,
			OldApproved:   oldApproved,
			NewApproved:   newApproved,
			Difference:    newApproved.Sub(oldApproved),
			ChangedReason: "rule segment, cap, or applicability changed",
		})
	}
	sort.SliceStable(impacts, func(i, j int) bool { return impacts[i].ClaimID < impacts[j].ClaimID })
	return impacts, nil
}
