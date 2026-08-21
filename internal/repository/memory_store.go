package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
)

type memoryData struct {
	projects          map[string]domain.GrantProject
	claimants         map[string]domain.Claimant
	claims            map[string]domain.ExpenseClaim
	idempotency       map[string]string
	receiptIndex      map[string]string
	rules             map[string]domain.RuleVersion
	ledgers           map[string]domain.AnnualLedger
	ledgerEntries     map[string][]domain.LedgerEntry
	settlements       map[string]domain.Settlement
	settlementByClaim map[string]string
	reviews           map[string][]domain.ReviewDecision
	audits            []domain.AuditEvent
}

func newMemoryData() *memoryData {
	return &memoryData{
		projects:          make(map[string]domain.GrantProject),
		claimants:         make(map[string]domain.Claimant),
		claims:            make(map[string]domain.ExpenseClaim),
		idempotency:       make(map[string]string),
		receiptIndex:      make(map[string]string),
		rules:             make(map[string]domain.RuleVersion),
		ledgers:           make(map[string]domain.AnnualLedger),
		ledgerEntries:     make(map[string][]domain.LedgerEntry),
		settlements:       make(map[string]domain.Settlement),
		settlementByClaim: make(map[string]string),
		reviews:           make(map[string][]domain.ReviewDecision),
		audits:            make([]domain.AuditEvent, 0),
	}
}

func (d *memoryData) clone() *memoryData {
	copyData := newMemoryData()
	for key, value := range d.projects {
		copyData.projects[key] = value
	}
	for key, value := range d.claimants {
		copyData.claimants[key] = value
	}
	for key, value := range d.claims {
		copyData.claims[key] = value
	}
	for key, value := range d.idempotency {
		copyData.idempotency[key] = value
	}
	for key, value := range d.receiptIndex {
		copyData.receiptIndex[key] = value
	}
	for key, value := range d.rules {
		value.Segments = append([]domain.RateSegment(nil), value.Segments...)
		copyData.rules[key] = value
	}
	for key, value := range d.ledgers {
		copyData.ledgers[key] = value
	}
	for key, entries := range d.ledgerEntries {
		copyData.ledgerEntries[key] = append([]domain.LedgerEntry(nil), entries...)
	}
	for key, value := range d.settlements {
		value.Explanation = append([]domain.ExplanationLine(nil), value.Explanation...)
		copyData.settlements[key] = value
	}
	for key, value := range d.settlementByClaim {
		copyData.settlementByClaim[key] = value
	}
	for key, values := range d.reviews {
		copyData.reviews[key] = append([]domain.ReviewDecision(nil), values...)
	}
	copyData.audits = append([]domain.AuditEvent(nil), d.audits...)
	return copyData
}

type MemoryStore struct {
	mu   sync.RWMutex
	data *memoryData
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: newMemoryData()}
}

func (s *MemoryStore) Seed(projects []domain.GrantProject, claimants []domain.Claimant, rules []domain.RuleVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, project := range projects {
		s.data.projects[project.ID] = project
	}
	for _, claimant := range claimants {
		s.data.claimants[claimant.ID] = claimant
	}
	for _, rule := range rules {
		s.data.rules[rule.ID] = rule
	}
}

func (s *MemoryStore) WithinTransaction(ctx context.Context, operation func(context.Context, application.Repositories) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	working := s.data.clone()
	repositories := &memoryRepositories{data: working}
	if err := operation(ctx, repositories); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.data = working
	return nil
}

type memoryRepositories struct {
	data *memoryData
}

func (s *MemoryStore) withRead(operation func(*memoryRepositories) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return operation(&memoryRepositories{data: s.data})
}

func (s *MemoryStore) withWrite(operation func(*memoryRepositories) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return operation(&memoryRepositories{data: s.data})
}

func (r *memoryRepositories) GetProject(_ context.Context, id string) (domain.GrantProject, error) {
	project, ok := r.data.projects[id]
	if !ok {
		return domain.GrantProject{}, fmt.Errorf("project %s: %w", id, domain.ErrNotFound)
	}
	return project, nil
}

func (s *MemoryStore) GetProject(ctx context.Context, id string) (domain.GrantProject, error) {
	var output domain.GrantProject
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetProject(ctx, id)
		return err
	})
	return output, err
}

func (r *memoryRepositories) GetClaimant(_ context.Context, id string) (domain.Claimant, error) {
	claimant, ok := r.data.claimants[id]
	if !ok {
		return domain.Claimant{}, fmt.Errorf("claimant %s: %w", id, domain.ErrNotFound)
	}
	return claimant, nil
}

func (s *MemoryStore) GetClaimant(ctx context.Context, id string) (domain.Claimant, error) {
	var output domain.Claimant
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetClaimant(ctx, id)
		return err
	})
	return output, err
}

func receiptKey(claimantID, projectID, digest string) string {
	return claimantID + "\x00" + projectID + "\x00" + digest
}

func ledgerKey(claimantID, projectID string, year int) string {
	return fmt.Sprintf("%s\x00%s\x00%d", claimantID, projectID, year)
}

func (r *memoryRepositories) CreateClaim(_ context.Context, claim domain.ExpenseClaim) error {
	if _, exists := r.data.claims[claim.ID]; exists {
		return fmt.Errorf("claim id %s: %w", claim.ID, domain.ErrConflict)
	}
	if _, exists := r.data.idempotency[claim.IdempotencyKey]; exists {
		return fmt.Errorf("idempotency key %s: %w", claim.IdempotencyKey, domain.ErrConflict)
	}
	key := receiptKey(claim.ClaimantID, claim.ProjectID, claim.ReceiptDigest)
	if _, exists := r.data.receiptIndex[key]; exists {
		return fmt.Errorf("receipt digest %s: %w", claim.ReceiptDigest, domain.ErrDuplicateClaim)
	}
	r.data.claims[claim.ID] = claim
	r.data.idempotency[claim.IdempotencyKey] = claim.ID
	r.data.receiptIndex[key] = claim.ID
	return nil
}

func (s *MemoryStore) CreateClaim(ctx context.Context, claim domain.ExpenseClaim) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.CreateClaim(ctx, claim) })
}

func (r *memoryRepositories) GetClaim(_ context.Context, id string) (domain.ExpenseClaim, error) {
	claim, ok := r.data.claims[id]
	if !ok {
		return domain.ExpenseClaim{}, fmt.Errorf("claim %s: %w", id, domain.ErrNotFound)
	}
	return claim, nil
}

func (s *MemoryStore) GetClaim(ctx context.Context, id string) (domain.ExpenseClaim, error) {
	var output domain.ExpenseClaim
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetClaim(ctx, id)
		return err
	})
	return output, err
}

func (r *memoryRepositories) FindClaimByIdempotencyKey(_ context.Context, key string) (domain.ExpenseClaim, error) {
	id, ok := r.data.idempotency[key]
	if !ok {
		return domain.ExpenseClaim{}, domain.ErrNotFound
	}
	return r.GetClaim(context.Background(), id)
}

func (s *MemoryStore) FindClaimByIdempotencyKey(ctx context.Context, key string) (domain.ExpenseClaim, error) {
	var output domain.ExpenseClaim
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.FindClaimByIdempotencyKey(ctx, key)
		return err
	})
	return output, err
}

func (r *memoryRepositories) FindDuplicateClaim(_ context.Context, claimantID, projectID, digest string) (domain.ExpenseClaim, error) {
	id, ok := r.data.receiptIndex[receiptKey(claimantID, projectID, digest)]
	if !ok {
		return domain.ExpenseClaim{}, domain.ErrNotFound
	}
	return r.GetClaim(context.Background(), id)
}

func (s *MemoryStore) FindDuplicateClaim(ctx context.Context, claimantID, projectID, digest string) (domain.ExpenseClaim, error) {
	var output domain.ExpenseClaim
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.FindDuplicateClaim(ctx, claimantID, projectID, digest)
		return err
	})
	return output, err
}

func (r *memoryRepositories) UpdateClaim(_ context.Context, claim domain.ExpenseClaim, expectedVersion int64) error {
	current, ok := r.data.claims[claim.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expectedVersion {
		return domain.ErrConflict
	}
	if claim.Version != expectedVersion+1 {
		return fmt.Errorf("claim %s version must advance once: %w", claim.ID, domain.ErrConflict)
	}
	r.data.claims[claim.ID] = claim
	return nil
}

func (s *MemoryStore) UpdateClaim(ctx context.Context, claim domain.ExpenseClaim, expectedVersion int64) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.UpdateClaim(ctx, claim, expectedVersion) })
}

func (r *memoryRepositories) ListClaims(_ context.Context, request domain.PageRequest) (domain.Page[domain.ExpenseClaim], error) {
	items := make([]domain.ExpenseClaim, 0, len(r.data.claims))
	statuses := make(map[string]bool, len(request.Statuses))
	for _, status := range request.Statuses {
		statuses[status] = true
	}
	for _, claim := range r.data.claims {
		if len(statuses) > 0 && !statuses[string(claim.Status)] {
			continue
		}
		items = append(items, claim)
	}
	sort.SliceStable(items, func(i, j int) bool {
		less := false
		switch request.SortBy {
		case "occurred_on":
			less = items[i].OccurredOn.Before(items[j].OccurredOn)
		case "amount":
			less = items[i].Amount.LessThan(items[j].Amount)
		case "status":
			less = items[i].Status < items[j].Status
		default:
			less = items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		if request.SortOrder == "desc" {
			return !less && items[i].ID != items[j].ID
		}
		return less
	})
	total := len(items)
	start := request.Offset()
	if start > total {
		start = total
	}
	end := start + request.PageSize
	if end > total {
		end = total
	}
	return domain.Page[domain.ExpenseClaim]{Items: append([]domain.ExpenseClaim(nil), items[start:end]...), Page: request.Page, PageSize: request.PageSize, Total: total}, nil
}

func (s *MemoryStore) ListClaims(ctx context.Context, request domain.PageRequest) (domain.Page[domain.ExpenseClaim], error) {
	var output domain.Page[domain.ExpenseClaim]
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListClaims(ctx, request)
		return err
	})
	return output, err
}

var _ application.Repositories = (*MemoryStore)(nil)
var _ application.UnitOfWork = (*MemoryStore)(nil)
var _ = errors.Is
