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

func TestSuccessfulReviewNotifiesOnceAfterCommit(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "1400.00", "review-success", "review-key-success")
	notifier := platform.NewLocalNotifier(f.clock)
	service := application.NewReviewService(f.store, f.clock, f.ids, notifier)
	reviewed, err := service.Decide(context.Background(), application.ReviewInput{
		ClaimID: claim.ID, ActorID: "manager-1", ActorRole: domain.RoleManager,
		Action: domain.ReviewReopen, Reason: "recheck supporting evidence",
		ExpectedVersion: claim.Version, RequestID: "request-review-success",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if reviewed.Status != domain.ClaimUnderReview || reviewed.Version != claim.Version+1 {
		t.Fatalf("reviewed claim not advanced: %+v", reviewed)
	}
	records := notifier.Records()
	if len(records) != 1 {
		t.Fatalf("got %d notifications, want exactly 1", len(records))
	}
	if got := records[0].Values["status"]; got != string(domain.ClaimUnderReview) {
		t.Fatalf("notification status = %q, want %q", got, domain.ClaimUnderReview)
	}
}

type txMarkerKey struct{}

type txMarkingUnit struct{ base application.UnitOfWork }

func (u txMarkingUnit) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	return u.base.WithinTransaction(ctx, func(txCtx context.Context, repositories application.Repositories) error {
		return operation(context.WithValue(txCtx, txMarkerKey{}, true), repositories)
	})
}

type contextInspectingNotifier struct {
	inner    application.Notifier
	sawTxCtx bool
}

func (n *contextInspectingNotifier) Send(ctx context.Context, notification application.Notification) error {
	if ctx.Value(txMarkerKey{}) != nil {
		n.sawTxCtx = true
	}
	return n.inner.Send(ctx, notification)
}

func TestSuccessfulReviewDeliversOutsideTransactionContext(t *testing.T) {
	f := newFixture(t)
	claim := f.createApprovedClaim(t, "1400.00", "review-ctx", "review-key-ctx")
	wrapped := platform.NewLocalNotifier(f.clock)
	notifier := &contextInspectingNotifier{inner: wrapped}
	service := application.NewReviewService(txMarkingUnit{base: f.store}, f.clock, f.ids, notifier)
	if _, err := service.Decide(context.Background(), application.ReviewInput{
		ClaimID: claim.ID, ActorID: "manager-1", ActorRole: domain.RoleManager,
		Action: domain.ReviewReopen, Reason: "recheck supporting evidence",
		ExpectedVersion: claim.Version, RequestID: "request-review-ctx",
	}); err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if notifier.sawTxCtx {
		t.Fatal("notification was delivered inside the transaction context")
	}
	if records := wrapped.Records(); len(records) != 1 {
		t.Fatalf("got %d notifications, want 1", len(records))
	}
}
