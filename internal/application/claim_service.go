package application

import (
	"context"
	"errors"
	"fmt"
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
	if err := ctx.Err(); err != nil {
		return domain.ExpenseClaim{}, false, err
	}
	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" {
		return domain.ExpenseClaim{}, false, domain.ValidationError(domain.FieldViolation{Field: "idempotency_key", Message: "required"})
	}
	existing, err := s.repositories.FindClaimByIdempotencyKey(ctx, key)
	if err == nil {
		return existing, true, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.ExpenseClaim{}, false, fmt.Errorf("lookup idempotency key: %w", err)
	}
	claimant, err := s.repositories.GetClaimant(ctx, input.ClaimantID)
	if err != nil {
		return domain.ExpenseClaim{}, false, fmt.Errorf("load claimant: %w", err)
	}
	if !claimant.Active {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError("CLAIMANT_INACTIVE", "claimant is inactive", domain.ErrForbidden)
	}
	project, err := s.repositories.GetProject(ctx, input.ProjectID)
	if err != nil {
		return domain.ExpenseClaim{}, false, fmt.Errorf("load project: %w", err)
	}
	if input.OccurredOn.Year() != project.Year {
		return domain.ExpenseClaim{}, false, domain.ValidationError(domain.FieldViolation{Field: "occurred_on", Message: "must fall in project year"})
	}
	_, err = s.repositories.FindDuplicateClaim(ctx, claimant.ID, project.ID, input.ReceiptDigest)
	if err == nil {
		return domain.ExpenseClaim{}, false, domain.NewBusinessError("DUPLICATE_CLAIM", "receipt was already submitted", domain.ErrDuplicateClaim)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.ExpenseClaim{}, false, fmt.Errorf("check duplicate receipt: %w", err)
	}
	claim, err := domain.NewExpenseClaim(
		s.ids.NewID("claim"), claimant.ID, project.ID, input.Category,
		input.ReceiptSummary, input.ReceiptDigest, input.OccurredOn,
		input.Amount, key, s.clock.Now(),
	)
	if err != nil {
		return domain.ExpenseClaim{}, false, err
	}
	if err := s.repositories.CreateClaim(ctx, claim); err != nil {
		return domain.ExpenseClaim{}, false, fmt.Errorf("create claim: %w", err)
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
