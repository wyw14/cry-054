package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type RateSegment struct {
	Threshold Money           `json:"threshold"`
	Rate      decimal.Decimal `json:"rate"`
}

type RuleConditions struct {
	Categories map[string]bool `json:"categories"`
	PlanCodes  map[string]bool `json:"plan_codes"`
	MinAmount  Money           `json:"min_amount"`
}

type RuleVersion struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"project_id"`
	Version       int            `json:"version"`
	EffectiveFrom time.Time      `json:"effective_from"`
	EffectiveTo   *time.Time     `json:"effective_to,omitempty"`
	Cap           Money          `json:"cap"`
	Segments      []RateSegment  `json:"segments"`
	Conditions    RuleConditions `json:"conditions"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
}

func (r RuleVersion) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.ProjectID) == "" || r.Version < 1 {
		return fmt.Errorf("rule identity: %w", ErrInvalidInput)
	}
	if !r.Cap.Positive() || len(r.Segments) == 0 {
		return fmt.Errorf("rule cap or segments: %w", ErrInvalidInput)
	}
	previous := ZeroMoney
	for index, segment := range r.Segments {
		if !segment.Threshold.Positive() || segment.Threshold.LessThanOrEqual(previous) {
			return fmt.Errorf("segment %d threshold must increase: %w", index, ErrInvalidInput)
		}
		if segment.Rate.IsNegative() || segment.Rate.GreaterThan(decimal.NewFromInt(1)) {
			return fmt.Errorf("segment %d rate outside [0,1]: %w", index, ErrInvalidInput)
		}
		previous = segment.Threshold
	}
	if r.EffectiveTo != nil && r.EffectiveTo.Before(r.EffectiveFrom) {
		return fmt.Errorf("rule effective interval: %w", ErrInvalidInput)
	}
	return nil
}

func (r RuleVersion) Applies(claim ExpenseClaim, planCode string) bool {
	date := dateOnly(claim.OccurredOn)
	if date.Before(dateOnly(r.EffectiveFrom)) {
		return false
	}
	if r.EffectiveTo != nil && date.After(dateOnly(*r.EffectiveTo)) {
		return false
	}
	if len(r.Conditions.Categories) > 0 && !r.Conditions.Categories[claim.Category] {
		return false
	}
	if len(r.Conditions.PlanCodes) > 0 && !r.Conditions.PlanCodes[planCode] {
		return false
	}
	return !claim.Amount.LessThan(r.Conditions.MinAmount)
}

func (r RuleVersion) Published() bool {
	return r.PublishedAt != nil
}

func (r RuleVersion) Publish(now time.Time) (RuleVersion, error) {
	if r.PublishedAt != nil {
		return RuleVersion{}, fmt.Errorf("rule %s is immutable after publication: %w", r.ID, ErrConflict)
	}
	if err := r.Validate(); err != nil {
		return RuleVersion{}, err
	}
	copyRule := r
	published := now.UTC()
	copyRule.PublishedAt = &published
	copyRule.Segments = append([]RateSegment(nil), r.Segments...)
	return copyRule, nil
}

func SelectRule(rules []RuleVersion, claim ExpenseClaim, planCode string) (RuleVersion, error) {
	candidates := make([]RuleVersion, 0, len(rules))
	for _, rule := range rules {
		if rule.Published() && rule.Applies(claim, planCode) {
			candidates = append(candidates, rule)
		}
	}
	if len(candidates) == 0 {
		return RuleVersion{}, fmt.Errorf("claim %s has no matching rule: %w", claim.ID, ErrRuleNotApplicable)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].EffectiveFrom.Equal(candidates[j].EffectiveFrom) {
			return candidates[i].Version > candidates[j].Version
		}
		return candidates[i].EffectiveFrom.After(candidates[j].EffectiveFrom)
	})
	return candidates[0], nil
}
