package application

import (
	"context"
	"io"
	"time"

	"github.com/wyw14/cry-054/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(prefix string) string
}

type ProjectRepository interface {
	GetProject(ctx context.Context, id string) (domain.GrantProject, error)
	GetClaimant(ctx context.Context, id string) (domain.Claimant, error)
}

type ClaimRepository interface {
	CreateClaim(ctx context.Context, claim domain.ExpenseClaim) error
	GetClaim(ctx context.Context, id string) (domain.ExpenseClaim, error)
	FindClaimByIdempotencyKey(ctx context.Context, key string) (domain.ExpenseClaim, error)
	FindDuplicateClaim(ctx context.Context, claimantID, projectID, receiptDigest string) (domain.ExpenseClaim, error)
	UpdateClaim(ctx context.Context, claim domain.ExpenseClaim, expectedVersion int64) error
	ListClaims(ctx context.Context, request domain.PageRequest) (domain.Page[domain.ExpenseClaim], error)
}

type RuleRepository interface {
	CreateRule(ctx context.Context, rule domain.RuleVersion) error
	PublishRule(ctx context.Context, rule domain.RuleVersion) error
	ListRules(ctx context.Context, projectID string) ([]domain.RuleVersion, error)
	GetRule(ctx context.Context, id string) (domain.RuleVersion, error)
}

type LedgerRepository interface {
	GetLedger(ctx context.Context, claimantID, projectID string, year int) (domain.AnnualLedger, error)
	SaveLedger(ctx context.Context, ledger domain.AnnualLedger, expectedVersion int64) error
	AppendLedgerEntry(ctx context.Context, entry domain.LedgerEntry) error
	ListLedgerEntries(ctx context.Context, claimantID, projectID string, year int) ([]domain.LedgerEntry, error)
}

type SettlementRepository interface {
	CreateSettlement(ctx context.Context, settlement domain.Settlement) error
	GetSettlement(ctx context.Context, id string) (domain.Settlement, error)
	GetSettlementByClaim(ctx context.Context, claimID string) (domain.Settlement, error)
	UpdateSettlement(ctx context.Context, settlement domain.Settlement, expectedVersion int64) error
	ListSettlements(ctx context.Context, projectID string, year int) ([]domain.Settlement, error)
}

type ReviewRepository interface {
	AppendReview(ctx context.Context, decision domain.ReviewDecision) error
	ListReviews(ctx context.Context, claimID string) ([]domain.ReviewDecision, error)
}

type AuditRepository interface {
	AppendAudit(ctx context.Context, event domain.AuditEvent) error
	ListAudit(ctx context.Context, filter domain.AuditFilter, page domain.PageRequest) (domain.Page[domain.AuditEvent], error)
}

type Repositories interface {
	ProjectRepository
	ClaimRepository
	RuleRepository
	LedgerRepository
	SettlementRepository
	ReviewRepository
	AuditRepository
}

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, operation func(context.Context, Repositories) error) error
}

type Notification struct {
	RecipientID string
	Template    string
	Values      map[string]string
}

type Notifier interface {
	Send(ctx context.Context, notification Notification) error
}

type StoredAttachment struct {
	ID          string
	Path        string
	ContentType string
	Size        int64
	SHA256      string
}

type AttachmentStore interface {
	Save(ctx context.Context, name, contentType string, size int64, reader io.Reader) (StoredAttachment, error)
	Open(ctx context.Context, id string) (io.ReadCloser, StoredAttachment, error)
}

type ExportSink interface {
	Write(ctx context.Context, name, contentType string, reader io.Reader) (StoredAttachment, error)
}
