package domain_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/domain"
)

func TestCalculatePreviewExplainsSegmentsAndAnnualRemaining(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	project, err := domain.NewGrantProject("project-1", "医疗补助", 2026, domain.MustMoney("12000.00"), now)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := domain.NewExpenseClaim("claim-1", "person-1", project.ID, "medical", "summary", "digest", now, domain.MustMoney("15000.00"), "key", now)
	if err != nil {
		t.Fatal(err)
	}
	published := now.Add(-time.Hour)
	rule := domain.RuleVersion{
		ID: "rule-1", ProjectID: project.ID, Version: 1, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Cap: domain.MustMoney("20000.00"), PublishedAt: &published,
		Segments: []domain.RateSegment{
			{Threshold: domain.MustMoney("10000.00"), Rate: decimal.RequireFromString("0.80")},
			{Threshold: domain.MustMoney("20000.00"), Rate: decimal.RequireFromString("0.50")},
		},
		Conditions: domain.RuleConditions{Categories: map[string]bool{"medical": true}, PlanCodes: map[string]bool{"enhanced": true}, MinAmount: domain.MustMoney("1.00")},
	}
	ledger := domain.NewAnnualLedger(claim.ClaimantID, project.ID, "enhanced", 2026, now)
	if err := ledger.Occup(domain.MustMoney("3000.00"), project.AnnualLimit, 1, now); err != nil {
		t.Fatal(err)
	}
	preview, err := domain.CalculatePreview(claim, rule, project, ledger, now)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := preview.Approved.String(), "9000.00"; got != want {
		t.Fatalf("approved %s, want %s", got, want)
	}
	if len(preview.Lines) != 2 {
		t.Fatalf("got %d explanation lines, want 2", len(preview.Lines))
	}
	if got := preview.RemainingAfter.String(); got != "0.00" {
		t.Fatalf("remaining after %s, want 0.00", got)
	}
}
