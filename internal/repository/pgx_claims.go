package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry-054/internal/domain"
)

func (r *pgxRepositories) GetProject(ctx context.Context, id string) (domain.GrantProject, error) {
	var project domain.GrantProject
	var annualLimit string
	err := r.q.QueryRow(ctx, `
		SELECT id, name, year, annual_limit::text, version, created_at
		FROM grant_projects WHERE id=$1`, id,
	).Scan(&project.ID, &project.Name, &project.Year, &annualLimit, &project.Version, &project.CreatedAt)
	if err != nil {
		return domain.GrantProject{}, mapPgxError(err)
	}
	project.AnnualLimit, err = parseDatabaseMoney(annualLimit)
	return project, err
}

func (r *pgxRepositories) GetClaimant(ctx context.Context, id string) (domain.Claimant, error) {
	var claimant domain.Claimant
	err := r.q.QueryRow(ctx, `
		SELECT id, display_name, identity_digest, plan_code, active, created_at
		FROM claimants WHERE id=$1`, id,
	).Scan(&claimant.ID, &claimant.DisplayName, &claimant.IdentityDigest, &claimant.PlanCode, &claimant.Active, &claimant.CreatedAt)
	if err != nil {
		return domain.Claimant{}, mapPgxError(err)
	}
	return claimant, nil
}

func (r *pgxRepositories) CreateClaim(ctx context.Context, claim domain.ExpenseClaim) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO expense_claims(
			id, claimant_id, project_id, category, receipt_digest, occurred_on,
			amount, status, version, idempotency_key, created_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		claim.ID, claim.ClaimantID, claim.ProjectID, claim.Category, claim.ReceiptDigest,
		claim.OccurredOn, claim.Amount.String(), claim.Status, claim.Version,
		claim.IdempotencyKey, claim.CreatedAt,
	)
	return mapPgxError(err)
}

func scanClaim(row pgx.Row) (domain.ExpenseClaim, error) {
	var claim domain.ExpenseClaim
	var amount string
	err := row.Scan(
		&claim.ID, &claim.ClaimantID, &claim.ProjectID, &claim.Category,
		&claim.ReceiptDigest, &claim.OccurredOn, &amount, &claim.Status,
		&claim.Version, &claim.IdempotencyKey, &claim.CreatedAt,
	)
	if err != nil {
		return domain.ExpenseClaim{}, mapPgxError(err)
	}
	claim.Amount, err = parseDatabaseMoney(amount)
	claim.UpdatedAt = claim.CreatedAt
	return claim, err
}

const claimSelect = `
	SELECT id, claimant_id, project_id, category, receipt_digest, occurred_on,
	       amount::text, status, version, idempotency_key, created_at
	FROM expense_claims`

func (r *pgxRepositories) GetClaim(ctx context.Context, id string) (domain.ExpenseClaim, error) {
	return scanClaim(r.q.QueryRow(ctx, claimSelect+` WHERE id=$1`, id))
}

func (r *pgxRepositories) FindClaimByIdempotencyKey(ctx context.Context, key string) (domain.ExpenseClaim, error) {
	return scanClaim(r.q.QueryRow(ctx, claimSelect+` WHERE idempotency_key=$1`, key))
}

func (r *pgxRepositories) FindDuplicateClaim(ctx context.Context, claimantID, projectID, receiptDigest string) (domain.ExpenseClaim, error) {
	return scanClaim(r.q.QueryRow(ctx, claimSelect+` WHERE claimant_id=$1 AND project_id=$2 AND receipt_digest=$3`, claimantID, projectID, receiptDigest))
}

func (r *pgxRepositories) UpdateClaim(ctx context.Context, claim domain.ExpenseClaim, expectedVersion int64) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE expense_claims SET status=$1, version=$2
		WHERE id=$3 AND version=$4`, claim.Status, claim.Version, claim.ID, expectedVersion)
	if err != nil {
		return mapPgxError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func (r *pgxRepositories) ListClaims(ctx context.Context, request domain.PageRequest) (domain.Page[domain.ExpenseClaim], error) {
	allowedSort := map[string]string{
		"created_at":  "created_at",
		"occurred_on": "occurred_on",
		"amount":      "amount",
		"status":      "status",
	}
	sortColumn, ok := allowedSort[request.SortBy]
	if !ok {
		return domain.Page[domain.ExpenseClaim]{}, domain.ErrInvalidInput
	}
	order := strings.ToUpper(request.SortOrder)
	if order != "ASC" && order != "DESC" {
		return domain.Page[domain.ExpenseClaim]{}, domain.ErrInvalidInput
	}
	where := ""
	args := []any{}
	if len(request.Statuses) > 0 {
		where = " WHERE status = ANY($1)"
		args = append(args, request.Statuses)
	}
	var total int
	if err := r.q.QueryRow(ctx, `SELECT count(*) FROM expense_claims`+where, args...).Scan(&total); err != nil {
		return domain.Page[domain.ExpenseClaim]{}, mapPgxError(err)
	}
	args = append(args, request.PageSize, request.Offset())
	limitPosition := len(args) - 1
	query := claimSelect + where + fmt.Sprintf(" ORDER BY %s %s, id ASC LIMIT $%d OFFSET $%d", sortColumn, order, limitPosition, limitPosition+1)
	rows, err := r.q.Query(ctx, query, args...)
	if err != nil {
		return domain.Page[domain.ExpenseClaim]{}, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.ExpenseClaim, 0, request.PageSize)
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return domain.Page[domain.ExpenseClaim]{}, err
		}
		items = append(items, claim)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.ExpenseClaim]{}, mapPgxError(err)
	}
	return domain.Page[domain.ExpenseClaim]{Items: items, Page: request.Page, PageSize: request.PageSize, Total: total}, nil
}
