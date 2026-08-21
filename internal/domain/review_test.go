package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-054/internal/domain"
)

func TestApplyReviewEnforcesRoleAndState(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	claim, err := domain.NewExpenseClaim("claim-1", "person-1", "project-1", "medical", "summary", "digest", now, domain.MustMoney("100.00"), "key", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := claim.Transition(domain.ClaimUnderReview, 1, now); err != nil {
		t.Fatal(err)
	}
	decision := domain.ReviewDecision{ID: "review-1", ClaimID: claim.ID, ActorID: "reviewer-1", ActorRole: domain.RoleReviewer, Action: domain.ReviewException, Reason: "documented exception"}
	if err := domain.ApplyReview(&claim, decision, 2, now); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v, want forbidden", err)
	}
	decision.ActorRole = domain.RoleManager
	if err := domain.ApplyReview(&claim, decision, 2, now); err != nil {
		t.Fatal(err)
	}
	if claim.Status != domain.ClaimApproved || claim.Version != 3 {
		t.Fatalf("unexpected claim state: %+v", claim)
	}
}
