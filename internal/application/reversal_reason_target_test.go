package application_test

import (
	"context"
	"testing"

	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
)

func TestReversalUsesOneCanonicalReasonAcrossAllRecords(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "1800.00", "reason-consistency", "reason-claim-key")
	settlement, _, err := application.NewSettlementService(f.store, f.clock, f.ids).Confirm(context.Background(), application.ConfirmSettlementInput{
		ClaimID: claim.ID, ExpectedClaim: claim.Version, ExpectedLedger: 1,
		ActorID: "manager-1", RequestID: "request-confirm-reason", IdempotencyKey: "settle-reason-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	rawReason := "  duplicate   provider\tinvoice  "
	reversed, err := application.NewCorrectionService(f.store, f.clock, f.ids).Reverse(context.Background(), application.ReverseSettlementInput{
		SettlementID: settlement.ID, Reason: rawReason, ActorID: "manager-1",
		RequestID: "request-reverse-reason", Expected: settlement.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	const canonical = "duplicate provider invoice"
	if reversed.ReversalReason != canonical {
		t.Fatalf("settlement reason = %q, want %q", reversed.ReversalReason, canonical)
	}
	entries, err := f.store.ListLedgerEntries(context.Background(), claim.ClaimantID, claim.ProjectID, f.project.Year)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Kind != "release" || entries[1].Reason != canonical {
		t.Fatalf("release entry reason is inconsistent: %+v", entries)
	}
	audits, err := f.store.ListAudit(context.Background(), domain.AuditFilter{Action: "settlement.reverse", SubjectID: settlement.ID}, domain.PageRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(audits.Items) != 1 || audits.Items[0].Detail["reason"] != canonical {
		t.Fatalf("reversal audit reason is inconsistent: %+v", audits.Items)
	}
}
