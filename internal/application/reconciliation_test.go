package application_test

import (
	"context"
	"testing"

	"github.com/wyw14/cry-054/internal/application"
)

func TestReconciliationReplaysConfirmedLedgerDeterministically(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "10000.00", "digest-reconcile", "claim-key")
	settlements := application.NewSettlementService(f.store, f.clock, f.ids)
	if _, _, err := settlements.Confirm(context.Background(), application.ConfirmSettlementInput{
		ClaimID: claim.ID, ExpectedClaim: 1, ExpectedLedger: 1,
		ActorID: "manager", RequestID: "req", IdempotencyKey: "confirm",
	}); err != nil {
		t.Fatal(err)
	}
	report, err := application.NewReconciliationService(f.store).Build(context.Background(), f.claimant.ID, f.project.ID, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 0 || !report.EntryTotal.Equal(report.LedgerTotal) || !report.Deterministic {
		t.Fatalf("unexpected report: %+v", report)
	}
}
