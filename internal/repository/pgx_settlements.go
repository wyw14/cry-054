package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry-054/internal/domain"
)

func (r *pgxRepositories) GetLedger(ctx context.Context, claimantID, projectID string, year int) (domain.AnnualLedger, error) {
	var ledger domain.AnnualLedger
	var occupied string
	err := r.q.QueryRow(ctx, `
		SELECT l.claimant_id, l.project_id, c.plan_code, l.year,
		       l.occupied::text, l.version, now()
		FROM annual_ledgers l
		JOIN claimants c ON c.id=l.claimant_id
		WHERE l.claimant_id=$1 AND l.project_id=$2 AND l.year=$3
		FOR UPDATE`, claimantID, projectID, year,
	).Scan(&ledger.ClaimantID, &ledger.ProjectID, &ledger.PlanCode, &ledger.Year, &occupied, &ledger.Version, &ledger.UpdatedAt)
	if err == pgx.ErrNoRows {
		var planCode string
		if planErr := r.q.QueryRow(ctx, `SELECT plan_code FROM claimants WHERE id=$1`, claimantID).Scan(&planCode); planErr != nil {
			return domain.AnnualLedger{}, mapPgxError(planErr)
		}
		_, insertErr := r.q.Exec(ctx, `
			INSERT INTO annual_ledgers(claimant_id, project_id, year, occupied, version)
			VALUES($1,$2,$3,0,1) ON CONFLICT DO NOTHING`, claimantID, projectID, year)
		if insertErr != nil {
			return domain.AnnualLedger{}, mapPgxError(insertErr)
		}
		return r.GetLedger(ctx, claimantID, projectID, year)
	}
	if err != nil {
		return domain.AnnualLedger{}, mapPgxError(err)
	}
	ledger.Occupied, err = parseDatabaseMoney(occupied)
	return ledger, err
}

func (r *pgxRepositories) SaveLedger(ctx context.Context, ledger domain.AnnualLedger, expectedVersion int64) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE annual_ledgers SET occupied=$1, version=$2
		WHERE claimant_id=$3 AND project_id=$4 AND year=$5 AND version=$6`,
		ledger.Occupied.String(), ledger.Version, ledger.ClaimantID, ledger.ProjectID, ledger.Year, expectedVersion,
	)
	if err != nil {
		return mapPgxError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (r *pgxRepositories) AppendLedgerEntry(ctx context.Context, entry domain.LedgerEntry) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO ledger_entries(
			settlement_id, claim_id, kind, amount, balance, occurred_at, actor_id, reason
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		entry.SettlementID, entry.ClaimID, entry.Kind, entry.Amount.String(),
		entry.Balance.String(), entry.OccurredAt, entry.ActorID, entry.Reason,
	)
	return mapPgxError(err)
}

func (r *pgxRepositories) ListLedgerEntries(ctx context.Context, claimantID, projectID string, year int) ([]domain.LedgerEntry, error) {
	rows, err := r.q.Query(ctx, `
		SELECT e.settlement_id, e.claim_id, e.kind, e.amount::text,
		       e.balance::text, e.occurred_at, e.actor_id, e.reason
		FROM ledger_entries e
		JOIN expense_claims c ON c.id=e.claim_id
		WHERE c.claimant_id=$1 AND c.project_id=$2 AND EXTRACT(YEAR FROM c.occurred_on)=$3
		ORDER BY e.occurred_at, e.settlement_id`, claimantID, projectID, year)
	if err != nil {
		return nil, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.LedgerEntry, 0)
	for rows.Next() {
		var entry domain.LedgerEntry
		var amount, balance string
		if err := rows.Scan(&entry.SettlementID, &entry.ClaimID, &entry.Kind, &amount, &balance, &entry.OccurredAt, &entry.ActorID, &entry.Reason); err != nil {
			return nil, mapPgxError(err)
		}
		entry.Amount, err = parseDatabaseMoney(amount)
		if err != nil {
			return nil, err
		}
		entry.Balance, err = parseDatabaseMoney(balance)
		if err != nil {
			return nil, err
		}
		items = append(items, entry)
	}
	return items, mapPgxError(rows.Err())
}

func (r *pgxRepositories) CreateSettlement(ctx context.Context, settlement domain.Settlement) error {
	explanation, err := json.Marshal(settlement.Explanation)
	if err != nil {
		return err
	}
	_, err = r.q.Exec(ctx, `
		INSERT INTO settlements(
			id, claim_id, rule_version_id, approved_amount, status,
			explanation, version, confirmed_at, idempotency_key, ledger_version_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		settlement.ID, settlement.ClaimID, settlement.RuleVersionID,
		settlement.ApprovedAmount.String(), settlement.Status, explanation,
		settlement.Version, settlement.ConfirmedAt, settlement.IdempotencyKey,
		settlement.LedgerVersionAt,
	)
	return mapPgxError(err)
}

func scanSettlement(row pgx.Row) (domain.Settlement, error) {
	var settlement domain.Settlement
	var approved string
	var explanation []byte
	err := row.Scan(
		&settlement.ID, &settlement.ClaimID, &settlement.RuleVersionID,
		&approved, &settlement.Status, &explanation, &settlement.Version,
		&settlement.ConfirmedAt, &settlement.ReversalReason,
		&settlement.CorrectionOfID, &settlement.IdempotencyKey,
		&settlement.LedgerVersionAt,
	)
	if err != nil {
		return domain.Settlement{}, mapPgxError(err)
	}
	settlement.ApprovedAmount, err = parseDatabaseMoney(approved)
	if err != nil {
		return domain.Settlement{}, err
	}
	if err := json.Unmarshal(explanation, &settlement.Explanation); err != nil {
		return domain.Settlement{}, fmt.Errorf("decode settlement explanation: %w", err)
	}
	return settlement, nil
}

const settlementSelect = `
	SELECT id, claim_id, rule_version_id, approved_amount::text, status,
	       explanation, version, confirmed_at, COALESCE(reversal_reason,''),
	       COALESCE(correction_of_id,''), idempotency_key, ledger_version_at
	FROM settlements`

func (r *pgxRepositories) GetSettlement(ctx context.Context, id string) (domain.Settlement, error) {
	return scanSettlement(r.q.QueryRow(ctx, settlementSelect+` WHERE id=$1`, id))
}

func (r *pgxRepositories) GetSettlementByClaim(ctx context.Context, claimID string) (domain.Settlement, error) {
	return scanSettlement(r.q.QueryRow(ctx, settlementSelect+` WHERE claim_id=$1`, claimID))
}

func (r *pgxRepositories) UpdateSettlement(ctx context.Context, settlement domain.Settlement, expectedVersion int64) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE settlements SET status=$1, version=$2, reversal_reason=$3,
		       correction_of_id=$4
		WHERE id=$5 AND version=$6`, settlement.Status, settlement.Version,
		settlement.ReversalReason, settlement.CorrectionOfID, settlement.ID, expectedVersion)
	if err != nil {
		return mapPgxError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (r *pgxRepositories) ListSettlements(ctx context.Context, projectID string, year int) ([]domain.Settlement, error) {
	rows, err := r.q.Query(ctx, settlementSelect+`
		WHERE claim_id IN (
			SELECT id FROM expense_claims WHERE project_id=$1 AND EXTRACT(YEAR FROM occurred_on)=$2
		) ORDER BY confirmed_at, id`, projectID, year)
	if err != nil {
		return nil, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.Settlement, 0)
	for rows.Next() {
		settlement, err := scanSettlement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, settlement)
	}
	return items, mapPgxError(rows.Err())
}
