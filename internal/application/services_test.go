package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/platform"
	"github.com/wyw14/cry-054/internal/repository"
)

type fixture struct {
	store    *repository.MemoryStore
	clock    platform.FixedClock
	ids      *platform.SequenceIDGenerator
	project  domain.GrantProject
	claimant domain.Claimant
	rule     domain.RuleVersion
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	now := time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	clock := platform.FixedClock{Value: now}
	ids := &platform.SequenceIDGenerator{}
	project, err := domain.NewGrantProject("project-1", "年度医疗补助", 2026, domain.MustMoney("50000.00"), now)
	if err != nil {
		t.Fatal(err)
	}
	claimant, err := domain.NewClaimant("person-1", "申请人", "identity-digest", "enhanced", now)
	if err != nil {
		t.Fatal(err)
	}
	published := now.Add(-24 * time.Hour)
	rule := domain.RuleVersion{
		ID: "rule-v1", ProjectID: project.ID, Version: 1,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Cap:           domain.MustMoney("30000.00"), PublishedAt: &published,
		Segments: []domain.RateSegment{
			{Threshold: domain.MustMoney("10000.00"), Rate: decimal.RequireFromString("0.8")},
			{Threshold: domain.MustMoney("30000.00"), Rate: decimal.RequireFromString("0.5")},
		},
		Conditions: domain.RuleConditions{Categories: map[string]bool{"medical": true}, PlanCodes: map[string]bool{"enhanced": true}, MinAmount: domain.MustMoney("1.00")},
	}
	store := repository.NewMemoryStore()
	store.Seed([]domain.GrantProject{project}, []domain.Claimant{claimant}, []domain.RuleVersion{rule})
	return fixture{store: store, clock: clock, ids: ids, project: project, claimant: claimant, rule: rule}
}

func (f fixture) createApprovedClaim(t *testing.T, amount, digest, key string) domain.ExpenseClaim {
	t.Helper()
	claim, err := domain.NewExpenseClaim("claim-"+digest, f.claimant.ID, f.project.ID, "medical", "receipt", digest, f.clock.Now(), domain.MustMoney(amount), key, f.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	claim.Status = domain.ClaimApproved
	if err := f.store.CreateClaim(context.Background(), claim); err != nil {
		t.Fatal(err)
	}
	return claim
}

func TestClaimServiceCreatesOnceForIdempotencyKey(t *testing.T) {
	f := newFixture(t)
	service := application.NewClaimService(f.store, f.clock, f.ids)
	input := application.CreateClaimInput{
		ClaimantID: f.claimant.ID, ProjectID: f.project.ID, Category: "medical",
		ReceiptSummary: "outpatient", ReceiptDigest: "digest-1", OccurredOn: f.clock.Now(),
		Amount: domain.MustMoney("1000.00"), IdempotencyKey: "create-key-1",
	}
	first, replayed, err := service.Create(context.Background(), input)
	if err != nil || replayed {
		t.Fatalf("first create: claim=%+v replayed=%v err=%v", first, replayed, err)
	}
	second, replayed, err := service.Create(context.Background(), input)
	if err != nil || !replayed || second.ID != first.ID {
		t.Fatalf("replay: claim=%+v replayed=%v err=%v", second, replayed, err)
	}
	if got := f.store.SnapshotCounts()["claims"]; got != 1 {
		t.Fatalf("got %d claims, want 1", got)
	}
}

func TestSettlementConfirmAtomicallyUpdatesSettlementClaimAndLedger(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "15000.00", "digest-confirm", "claim-key")
	service := application.NewSettlementService(f.store, f.clock, f.ids)
	settlement, replayed, err := service.Confirm(context.Background(), application.ConfirmSettlementInput{
		ClaimID: claim.ID, ExpectedClaim: 1, ExpectedLedger: 1,
		ActorID: "manager-1", RequestID: "req-1", IdempotencyKey: "confirm-key",
	})
	if err != nil || replayed {
		t.Fatalf("confirm: settlement=%+v replayed=%v err=%v", settlement, replayed, err)
	}
	if got, want := settlement.ApprovedAmount.String(), "10500.00"; got != want {
		t.Fatalf("approved %s, want %s", got, want)
	}
	storedClaim, err := f.store.GetClaim(context.Background(), claim.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := f.store.GetLedger(context.Background(), claim.ClaimantID, claim.ProjectID, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if storedClaim.Status != domain.ClaimSettled || ledger.Occupied.String() != "10500.00" {
		t.Fatalf("claim=%s occupied=%s", storedClaim.Status, ledger.Occupied)
	}
	counts := f.store.SnapshotCounts()
	if counts["settlements"] != 1 || counts["audit_events"] != 1 {
		t.Fatalf("unexpected transaction counts: %+v", counts)
	}
}

func TestSettlementConflictRollsBackEverySideEffect(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "5000.00", "digest-conflict", "claim-key")
	service := application.NewSettlementService(f.store, f.clock, f.ids)
	_, _, err := service.Confirm(context.Background(), application.ConfirmSettlementInput{
		ClaimID: claim.ID, ExpectedClaim: 99, ExpectedLedger: 1,
		ActorID: "manager-1", RequestID: "req-2", IdempotencyKey: "confirm-key",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("got %v, want conflict", err)
	}
	storedClaim, err := f.store.GetClaim(context.Background(), claim.ID)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := f.store.GetLedger(context.Background(), claim.ClaimantID, claim.ProjectID, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if storedClaim.Status != domain.ClaimApproved || !ledger.Occupied.IsZero() {
		t.Fatalf("rollback failed: claim=%s occupied=%s", storedClaim.Status, ledger.Occupied)
	}
	counts := f.store.SnapshotCounts()
	if counts["settlements"] != 0 || counts["audit_events"] != 0 {
		t.Fatalf("rollback left side effects: %+v", counts)
	}
}
