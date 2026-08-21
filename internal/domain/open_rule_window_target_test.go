package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestOpenEndedPublishedRuleRemainsApplicableWithoutPanic(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	published := now.Add(-24 * time.Hour)
	rule := RuleVersion{
		ID: "open-rule", ProjectID: "project-open", Version: 3,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveTo:   nil,
		Cap:           MustMoney("10000.00"),
		Segments:      []RateSegment{{Threshold: MustMoney("10000.00"), Rate: decimal.RequireFromString("0.75")}},
		Conditions: RuleConditions{
			Categories: map[string]bool{"medical": true},
			PlanCodes:  map[string]bool{"enhanced": true},
			MinAmount:  MustMoney("1.00"),
		},
		PublishedAt: &published,
	}
	claim, err := NewExpenseClaim("claim-open", "person-open", "project-open", "medical", "receipt", "digest-open", now, MustMoney("1200.00"), "key-open", now)
	if err != nil {
		t.Fatal(err)
	}
	deferredReached := false
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("open-ended rule applicability panicked: %v", recovered)
		}
		if !deferredReached {
			t.Fatal("applicability evaluation did not complete")
		}
	}()
	project, err := NewGrantProject("project-open", "open grant", 2026, MustMoney("20000.00"), now)
	if err != nil {
		t.Fatal(err)
	}
	ledger := NewAnnualLedger("person-open", "project-open", "enhanced", 2026, now)
	preview, err := CalculatePreview(claim, rule, project, ledger, now)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Approved.Positive() {
		t.Fatal("open-ended published rule should approve an in-scope claim")
	}
	deferredReached = true
}
