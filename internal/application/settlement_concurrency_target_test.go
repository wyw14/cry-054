package application_test

import (
	"context"
	"sync"
	"testing"

	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
)

func TestConcurrentConfirmationsKeepRequestStateIsolated(t *testing.T) {
	f := newFixture(t)
	second, err := domain.NewClaimant("person-2", "second claimant", "identity-digest-2", "enhanced", f.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	f.store.Seed(nil, []domain.Claimant{second}, nil)
	firstClaim := f.createApprovedClaim(t, "4000.00", "race-first", "claim-key-first")
	secondClaim, err := domain.NewExpenseClaim("claim-race-second", second.ID, f.project.ID, "medical", "receipt", "race-second", f.clock.Now(), domain.MustMoney("5000.00"), "claim-key-second", f.clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	secondClaim.Status = domain.ClaimApproved
	if err := f.store.CreateClaim(context.Background(), secondClaim); err != nil {
		t.Fatal(err)
	}

	service := application.NewSettlementService(f.store, f.clock, f.ids)
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	errorsByClaim := make(chan error, 2)
	var workers sync.WaitGroup
	for _, item := range []struct {
		claimID string
		key     string
		request string
	}{{firstClaim.ID, "confirm-first", "request-first"}, {secondClaim.ID, "confirm-second", "request-second"}} {
		item := item
		workers.Add(1)
		go func() {
			defer workers.Done()
			ready <- struct{}{}
			<-start
			_, _, runErr := service.Confirm(context.Background(), application.ConfirmSettlementInput{
				ClaimID: item.claimID, ExpectedClaim: 1, ExpectedLedger: 1,
				ActorID: "manager", RequestID: item.request, IdempotencyKey: item.key,
			})
			errorsByClaim <- runErr
		}()
	}
	<-ready
	<-ready
	close(start)
	workers.Wait()
	close(errorsByClaim)
	for runErr := range errorsByClaim {
		if runErr != nil {
			t.Fatalf("concurrent confirmation failed: %v", runErr)
		}
	}
	if got := f.store.SnapshotCounts()["settlements"]; got != 2 {
		t.Fatalf("stored settlements = %d, want 2", got)
	}
}
