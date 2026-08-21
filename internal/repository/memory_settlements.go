package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/wyw14/cry-054/internal/domain"
)

func (r *memoryRepositories) GetLedger(_ context.Context, claimantID, projectID string, year int) (domain.AnnualLedger, error) {
	key := ledgerKey(claimantID, projectID, year)
	ledger, exists := r.data.ledgers[key]
	if !exists {
		claimant, claimantExists := r.data.claimants[claimantID]
		if !claimantExists {
			return domain.AnnualLedger{}, domain.ErrNotFound
		}
		ledger = domain.NewAnnualLedger(claimantID, projectID, claimant.PlanCode, year, claimant.CreatedAt)
		r.data.ledgers[key] = ledger
	}
	return ledger, nil
}

func (s *MemoryStore) GetLedger(ctx context.Context, claimantID, projectID string, year int) (domain.AnnualLedger, error) {
	var output domain.AnnualLedger
	err := s.withWrite(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetLedger(ctx, claimantID, projectID, year)
		return err
	})
	return output, err
}

func (r *memoryRepositories) SaveLedger(_ context.Context, ledger domain.AnnualLedger, expectedVersion int64) error {
	key := ledgerKey(ledger.ClaimantID, ledger.ProjectID, ledger.Year)
	current, exists := r.data.ledgers[key]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Version != expectedVersion || ledger.Version != expectedVersion+1 {
		return domain.ErrConflict
	}
	r.data.ledgers[key] = ledger
	return nil
}

func (s *MemoryStore) SaveLedger(ctx context.Context, ledger domain.AnnualLedger, expectedVersion int64) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.SaveLedger(ctx, ledger, expectedVersion) })
}

func (r *memoryRepositories) AppendLedgerEntry(_ context.Context, entry domain.LedgerEntry) error {
	claim, exists := r.data.claims[entry.ClaimID]
	if !exists {
		return domain.ErrNotFound
	}
	project, exists := r.data.projects[claim.ProjectID]
	if !exists {
		return domain.ErrNotFound
	}
	key := ledgerKey(claim.ClaimantID, claim.ProjectID, project.Year)
	r.data.ledgerEntries[key] = append(r.data.ledgerEntries[key], entry)
	return nil
}

func (s *MemoryStore) AppendLedgerEntry(ctx context.Context, entry domain.LedgerEntry) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.AppendLedgerEntry(ctx, entry) })
}

func (r *memoryRepositories) ListLedgerEntries(_ context.Context, claimantID, projectID string, year int) ([]domain.LedgerEntry, error) {
	items := append([]domain.LedgerEntry(nil), r.data.ledgerEntries[ledgerKey(claimantID, projectID, year)]...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].SettlementID < items[j].SettlementID
		}
		return items[i].OccurredAt.Before(items[j].OccurredAt)
	})
	return items, nil
}

func (s *MemoryStore) ListLedgerEntries(ctx context.Context, claimantID, projectID string, year int) ([]domain.LedgerEntry, error) {
	var output []domain.LedgerEntry
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListLedgerEntries(ctx, claimantID, projectID, year)
		return err
	})
	return output, err
}

func (r *memoryRepositories) CreateSettlement(_ context.Context, settlement domain.Settlement) error {
	if _, exists := r.data.settlements[settlement.ID]; exists {
		return domain.ErrConflict
	}
	if _, exists := r.data.settlementByClaim[settlement.ClaimID]; exists {
		return domain.ErrConflict
	}
	r.data.settlements[settlement.ID] = settlement
	r.data.settlementByClaim[settlement.ClaimID] = settlement.ID
	return nil
}

func (s *MemoryStore) CreateSettlement(ctx context.Context, settlement domain.Settlement) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.CreateSettlement(ctx, settlement) })
}

func (r *memoryRepositories) GetSettlement(_ context.Context, id string) (domain.Settlement, error) {
	settlement, exists := r.data.settlements[id]
	if !exists {
		return domain.Settlement{}, domain.ErrNotFound
	}
	settlement.Explanation = append([]domain.ExplanationLine(nil), settlement.Explanation...)
	return settlement, nil
}

func (s *MemoryStore) GetSettlement(ctx context.Context, id string) (domain.Settlement, error) {
	var output domain.Settlement
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetSettlement(ctx, id)
		return err
	})
	return output, err
}

func (r *memoryRepositories) GetSettlementByClaim(ctx context.Context, claimID string) (domain.Settlement, error) {
	id, exists := r.data.settlementByClaim[claimID]
	if !exists {
		return domain.Settlement{}, domain.ErrNotFound
	}
	return r.GetSettlement(ctx, id)
}

func (s *MemoryStore) GetSettlementByClaim(ctx context.Context, claimID string) (domain.Settlement, error) {
	var output domain.Settlement
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetSettlementByClaim(ctx, claimID)
		return err
	})
	return output, err
}

func (r *memoryRepositories) UpdateSettlement(_ context.Context, settlement domain.Settlement, expectedVersion int64) error {
	current, exists := r.data.settlements[settlement.ID]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Version != expectedVersion || settlement.Version != expectedVersion+1 {
		return domain.ErrConflict
	}
	r.data.settlements[settlement.ID] = settlement
	return nil
}

func (s *MemoryStore) UpdateSettlement(ctx context.Context, settlement domain.Settlement, expectedVersion int64) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.UpdateSettlement(ctx, settlement, expectedVersion) })
}

func (r *memoryRepositories) ListSettlements(_ context.Context, projectID string, year int) ([]domain.Settlement, error) {
	items := make([]domain.Settlement, 0)
	for _, settlement := range r.data.settlements {
		claim, exists := r.data.claims[settlement.ClaimID]
		if !exists || claim.ProjectID != projectID || claim.OccurredOn.Year() != year {
			continue
		}
		items = append(items, settlement)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *MemoryStore) ListSettlements(ctx context.Context, projectID string, year int) ([]domain.Settlement, error) {
	var output []domain.Settlement
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListSettlements(ctx, projectID, year)
		return err
	})
	return output, err
}

func (r *memoryRepositories) AppendReview(_ context.Context, decision domain.ReviewDecision) error {
	r.data.reviews[decision.ClaimID] = append(r.data.reviews[decision.ClaimID], decision)
	return nil
}

func (s *MemoryStore) AppendReview(ctx context.Context, decision domain.ReviewDecision) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.AppendReview(ctx, decision) })
}

func (r *memoryRepositories) ListReviews(_ context.Context, claimID string) ([]domain.ReviewDecision, error) {
	return append([]domain.ReviewDecision(nil), r.data.reviews[claimID]...), nil
}

func (s *MemoryStore) ListReviews(ctx context.Context, claimID string) ([]domain.ReviewDecision, error) {
	var output []domain.ReviewDecision
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListReviews(ctx, claimID)
		return err
	})
	return output, err
}

func (r *memoryRepositories) AppendAudit(_ context.Context, event domain.AuditEvent) error {
	event.ID = int64(len(r.data.audits) + 1)
	r.data.audits = append(r.data.audits, event)
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.AppendAudit(ctx, event) })
}

func (r *memoryRepositories) ListAudit(_ context.Context, filter domain.AuditFilter, page domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	items := make([]domain.AuditEvent, 0)
	for _, event := range r.data.audits {
		if filter.ActorID != "" && event.ActorID != filter.ActorID {
			continue
		}
		if filter.SubjectType != "" && event.SubjectType != filter.SubjectType {
			continue
		}
		if filter.SubjectID != "" && event.SubjectID != filter.SubjectID {
			continue
		}
		if filter.Action != "" && event.Action != filter.Action {
			continue
		}
		if filter.From != nil && event.CreatedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && event.CreatedAt.After(*filter.To) {
			continue
		}
		items = append(items, event)
	}
	total := len(items)
	start := page.Offset()
	if start > total {
		start = total
	}
	end := start + page.PageSize
	if end > total {
		end = total
	}
	return domain.Page[domain.AuditEvent]{Items: append([]domain.AuditEvent(nil), items[start:end]...), Page: page.Page, PageSize: page.PageSize, Total: total}, nil
}

func (s *MemoryStore) ListAudit(ctx context.Context, filter domain.AuditFilter, page domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	var output domain.Page[domain.AuditEvent]
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListAudit(ctx, filter, page)
		return err
	})
	return output, err
}

func (s *MemoryStore) SnapshotCounts() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]int{
		"projects":     len(s.data.projects),
		"claimants":    len(s.data.claimants),
		"claims":       len(s.data.claims),
		"rules":        len(s.data.rules),
		"ledgers":      len(s.data.ledgers),
		"settlements":  len(s.data.settlements),
		"reviews":      len(s.data.reviews),
		"audit_events": len(s.data.audits),
	}
}

var _ = fmt.Sprintf
