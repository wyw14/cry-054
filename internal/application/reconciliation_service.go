package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/wyw14/cry-054/internal/domain"
)

type ReconciliationIssue struct {
	Code         string       `json:"code"`
	SettlementID string       `json:"settlement_id,omitempty"`
	Expected     domain.Money `json:"expected"`
	Actual       domain.Money `json:"actual"`
	Message      string       `json:"message"`
}

type ReconciliationReport struct {
	ClaimantID    string                `json:"claimant_id"`
	ProjectID     string                `json:"project_id"`
	Year          int                   `json:"year"`
	LedgerTotal   domain.Money          `json:"ledger_total"`
	EntryTotal    domain.Money          `json:"entry_total"`
	Issues        []ReconciliationIssue `json:"issues"`
	Deterministic bool                  `json:"deterministic"`
}

type ReconciliationService struct {
	repositories Repositories
}

func NewReconciliationService(repositories Repositories) *ReconciliationService {
	return &ReconciliationService{repositories: repositories}
}

func (s *ReconciliationService) Build(ctx context.Context, claimantID, projectID string, year int) (ReconciliationReport, error) {
	ledger, err := s.repositories.GetLedger(ctx, claimantID, projectID, year)
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("load ledger for reconciliation: %w", err)
	}
	entries, err := s.repositories.ListLedgerEntries(ctx, claimantID, projectID, year)
	if err != nil {
		return ReconciliationReport{}, fmt.Errorf("list ledger entries: %w", err)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].OccurredAt.Equal(entries[j].OccurredAt) {
			return entries[i].SettlementID < entries[j].SettlementID
		}
		return entries[i].OccurredAt.Before(entries[j].OccurredAt)
	})
	running := domain.ZeroMoney
	issues := make([]ReconciliationIssue, 0)
	seenSettlements := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if seenSettlements[entry.SettlementID] && entry.Kind == "occupy" {
			issues = append(issues, ReconciliationIssue{Code: "DUPLICATE_OCCUPANCY", SettlementID: entry.SettlementID, Message: "settlement occupied annual allowance more than once"})
		}
		seenSettlements[entry.SettlementID] = true
		switch entry.Kind {
		case "occupy":
			running = running.Add(entry.Amount)
		case "release":
			running = running.Sub(entry.Amount)
		default:
			issues = append(issues, ReconciliationIssue{Code: "UNKNOWN_ENTRY_KIND", SettlementID: entry.SettlementID, Message: "ledger entry kind is unknown"})
		}
		if !entry.Balance.Equal(running) {
			issues = append(issues, ReconciliationIssue{
				Code:         "RUNNING_BALANCE_MISMATCH",
				SettlementID: entry.SettlementID,
				Expected:     running,
				Actual:       entry.Balance,
				Message:      "stored running balance differs from replayed entries",
			})
		}
	}
	if !running.Equal(ledger.Occupied) {
		issues = append(issues, ReconciliationIssue{
			Code:     "LEDGER_TOTAL_MISMATCH",
			Expected: running,
			Actual:   ledger.Occupied,
			Message:  "annual ledger total differs from replayed entries",
		})
	}
	return ReconciliationReport{
		ClaimantID:    claimantID,
		ProjectID:     projectID,
		Year:          year,
		LedgerTotal:   ledger.Occupied,
		EntryTotal:    running,
		Issues:        issues,
		Deterministic: true,
	}, nil
}
