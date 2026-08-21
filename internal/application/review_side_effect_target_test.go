package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/platform"
)

type auditFailureRepositories struct{ application.Repositories }

func (auditFailureRepositories) AppendAudit(context.Context, domain.AuditEvent) error {
	return errors.New("audit sink unavailable")
}

type auditFailureUnit struct{ base application.UnitOfWork }

func (u auditFailureUnit) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	return u.base.WithinTransaction(ctx, func(txCtx context.Context, repositories application.Repositories) error {
		return operation(txCtx, auditFailureRepositories{Repositories: repositories})
	})
}

func TestFailedReviewTransactionDoesNotLeakNotification(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "1400.00", "review-rollback", "review-key")
	notifier := platform.NewLocalNotifier(f.clock)
	service := application.NewReviewService(auditFailureUnit{base: f.store}, f.clock, f.ids, notifier)
	_, err := service.Decide(context.Background(), application.ReviewInput{
		ClaimID: claim.ID, ActorID: "manager-1", ActorRole: domain.RoleManager,
		Action: domain.ReviewReopen, Reason: "recheck supporting evidence",
		ExpectedVersion: claim.Version, RequestID: "request-review-rollback",
	})
	if err == nil {
		t.Fatal("expected review transaction to fail when audit append fails")
	}
	stored, getErr := f.store.GetClaim(context.Background(), claim.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Status != domain.ClaimApproved || stored.Version != claim.Version {
		t.Fatalf("claim was not rolled back: %+v", stored)
	}
	if records := notifier.Records(); len(records) != 0 {
		t.Fatalf("notification escaped failed transaction: %+v", records)
	}
}
