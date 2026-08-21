package application_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/application"
)

func TestPublishedRuleSegmentsCannotBeMutatedThroughListResult(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "5000.00", "slice-isolation", "slice-key")
	listed, err := f.store.ListRules(context.Background(), f.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || len(listed[0].Segments) == 0 {
		t.Fatalf("unexpected rules: %+v", listed)
	}
	listed[0].Segments[0].Rate = decimal.Zero
	listed[0].Segments[0].Threshold = listed[0].Segments[0].Threshold.Add(f.rule.Cap)

	preview, err := application.NewPreviewService(f.store, f.clock).Preview(context.Background(), claim.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := preview.Approved.String(), "4000.00"; got != want {
		t.Fatalf("approved amount after caller mutation = %s, want %s", got, want)
	}
}
