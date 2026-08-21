package domain_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/domain"
)

func TestSelectRuleKeepsPublishedHistoryAndChoosesApplicableVersion(t *testing.T) {
	published := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	claim, err := domain.NewExpenseClaim(
		"claim-1", "person-1", "project-1", "medical", "receipt", "digest",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), domain.MustMoney("5000.00"),
		"key-1", published,
	)
	if err != nil {
		t.Fatal(err)
	}
	makeRule := func(id string, version int, effective time.Time) domain.RuleVersion {
		return domain.RuleVersion{
			ID: id, ProjectID: "project-1", Version: version, EffectiveFrom: effective,
			Cap:         domain.MustMoney("10000.00"),
			Segments:    []domain.RateSegment{{Threshold: domain.MustMoney("10000.00"), Rate: decimal.RequireFromString("0.8")}},
			Conditions:  domain.RuleConditions{Categories: map[string]bool{"medical": true}, PlanCodes: map[string]bool{"enhanced": true}, MinAmount: domain.MustMoney("1.00")},
			PublishedAt: &published,
		}
	}
	rules := []domain.RuleVersion{
		makeRule("v1", 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		makeRule("v2", 2, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
	}
	selected, err := domain.SelectRule(rules, claim, "enhanced")
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != "v2" {
		t.Fatalf("selected %s, want v2", selected.ID)
	}
	if _, err := rules[0].Publish(published); err == nil {
		t.Fatal("published historical rule should be immutable")
	}
}
