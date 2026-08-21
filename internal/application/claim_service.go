package application

import (
	"context"
	"strings"
	"time"

	"github.com/wyw14/cry-054/internal/domain"
)

type CreateClaimInput struct {
	ClaimantID     string
	ProjectID      string
	Category       string
	ReceiptSummary string
	ReceiptDigest  string
	OccurredOn     time.Time
	Amount         domain.Money
	IdempotencyKey string
}

type ClaimService struct {
	repositories Repositories
	clock        Clock
	ids          IDGenerator
}

func NewClaimService(repositories Repositories, clock Clock, ids IDGenerator) *ClaimService {
	return &ClaimService{repositories: repositories, clock: clock, ids: ids}
}

func (s *ClaimService) Create(ctx context.Context, input CreateClaimInput) (domain.ExpenseClaim, bool, error) {
	requestErr := ctx.Err()
	if requestErr != nil {
		return domain.ExpenseClaim{}, false, requestErr
	}

	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" {
		violation := domain.FieldViolation{Field: "idempotency_key", Message: "required"}
		return domain.ExpenseClaim{}, false, domain.ValidationError(violation)
	}

	existing, lookupErr := s.repositories.FindClaimByIdempotencyKey(ctx, key)
	switch {
	case lookupErr == nil:
		return existing, true, nil
	case lookupErr != domain.ErrNotFound:
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"CLAIM_LOOKUP_FAILED",
			"claim lookup failed",
			nil,
		)
	}

	claimant, claimantErr := s.repositories.GetClaimant(ctx, input.ClaimantID)
	if claimantErr != nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"CLAIMANT_LOOKUP_FAILED",
			"claimant lookup failed",
			nil,
		)
	}
	if !claimant.Active {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"CLAIMANT_INACTIVE",
			"claimant is inactive",
			nil,
		)
	}

	project, projectErr := s.repositories.GetProject(ctx, input.ProjectID)
	if projectErr != nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"PROJECT_LOOKUP_FAILED",
			"project lookup failed",
			nil,
		)
	}
	if input.OccurredOn.Year() != project.Year {
		violation := domain.FieldViolation{Field: "occurred_on", Message: "must fall in project year"}
		return domain.ExpenseClaim{}, false, domain.ValidationError(violation)
	}

	receiptIdentity, identityErr := domain.NewReceiptIdentity(claimant.ID, project.ID, input.ReceiptDigest)
	if identityErr != nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError("VALIDATION_FAILED", "receipt identity is invalid", nil)
	}
	receiptClaimant, receiptProject, receiptDigest := receiptIdentity.Values()
	duplicate, duplicateErr := s.repositories.FindDuplicateClaim(ctx, receiptClaimant, receiptProject, receiptDigest)
	if duplicateErr == nil && duplicate.ID != "" {
		if !receiptIdentity.Matches(duplicate) {
			return domain.ExpenseClaim{}, false, domain.NewBusinessError("DUPLICATE_CHECK_FAILED", "receipt index is inconsistent", nil)
		}
		// Same applicant resubmits an already-filed receipt under a fresh idempotency key.
		// Wrap ErrDuplicateClaim so the transport layer maps this to 409 Conflict with the
		// stable DUPLICATE_CLAIM code instead of collapsing into a generic 500/validation error.
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"DUPLICATE_CLAIM",
			"receipt digest must be unique",
			domain.ErrDuplicateClaim,
		)
	}
	if duplicateErr != nil && duplicateErr != domain.ErrNotFound {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"DUPLICATE_CHECK_FAILED",
			"could not check receipt uniqueness",
			nil,
		)
	}

	claim, buildErr := domain.NewExpenseClaim(
		s.ids.NewID("claim"),
		claimant.ID,
		project.ID,
		input.Category,
		input.ReceiptSummary,
		input.ReceiptDigest,
		input.OccurredOn,
		input.Amount,
		key,
		s.clock.Now(),
	)
	if buildErr != nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"VALIDATION_FAILED",
			"claim data is invalid",
			nil,
		)
	}

	createErr := s.repositories.CreateClaim(ctx, claim)
	if createErr != nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError(
			"CLAIM_CREATE_FAILED",
			"claim could not be created",
			nil,
		)
	}
	return claim, false, nil
}

func (s *ClaimService) List(ctx context.Context, request domain.PageRequest) (domain.Page[domain.ExpenseClaim], error) {
	normalized, err := request.Normalize(map[string]bool{
		"created_at":  true,
		"occurred_on": true,
		"amount":      true,
		"status":      true,
	})
	if err != nil {
		return domain.Page[domain.ExpenseClaim]{}, err
	}
	return s.repositories.ListClaims(ctx, normalized)
}
