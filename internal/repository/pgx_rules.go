package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry-054/internal/domain"
)

func (r *pgxRepositories) CreateRule(ctx context.Context, rule domain.RuleVersion) error {
	segments, err := json.Marshal(rule.Segments)
	if err != nil {
		return fmt.Errorf("marshal rule segments: %w", err)
	}
	conditions, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("marshal rule conditions: %w", err)
	}
	_, err = r.q.Exec(ctx, `
		INSERT INTO rule_versions(
			id, project_id, version, effective_from, effective_to, cap,
			segments, conditions, published_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		rule.ID, rule.ProjectID, rule.Version, rule.EffectiveFrom, rule.EffectiveTo,
		rule.Cap.String(), segments, conditions, rule.PublishedAt,
	)
	return mapPgxError(err)
}

func (r *pgxRepositories) PublishRule(ctx context.Context, rule domain.RuleVersion) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE rule_versions SET published_at=$1
		WHERE id=$2 AND published_at IS NULL`, rule.PublishedAt, rule.ID)
	if err != nil {
		return mapPgxError(err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func scanRule(row pgx.Row) (domain.RuleVersion, error) {
	var rule domain.RuleVersion
	var cap string
	var segments []byte
	var conditions []byte
	err := row.Scan(
		&rule.ID, &rule.ProjectID, &rule.Version, &rule.EffectiveFrom,
		&rule.EffectiveTo, &cap, &segments, &conditions, &rule.PublishedAt,
	)
	if err != nil {
		return domain.RuleVersion{}, mapPgxError(err)
	}
	rule.Cap, err = parseDatabaseMoney(cap)
	if err != nil {
		return domain.RuleVersion{}, err
	}
	if err := json.Unmarshal(segments, &rule.Segments); err != nil {
		return domain.RuleVersion{}, fmt.Errorf("decode rule segments: %w", err)
	}
	if err := json.Unmarshal(conditions, &rule.Conditions); err != nil {
		return domain.RuleVersion{}, fmt.Errorf("decode rule conditions: %w", err)
	}
	return rule, nil
}

const ruleSelect = `
	SELECT id, project_id, version, effective_from, effective_to, cap::text,
	       segments, conditions, published_at
	FROM rule_versions`

func (r *pgxRepositories) ListRules(ctx context.Context, projectID string) ([]domain.RuleVersion, error) {
	rows, err := r.q.Query(ctx, ruleSelect+` WHERE project_id=$1 ORDER BY version ASC`, projectID)
	if err != nil {
		return nil, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.RuleVersion, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, rule)
	}
	return items, mapPgxError(rows.Err())
}

func (r *pgxRepositories) GetRule(ctx context.Context, id string) (domain.RuleVersion, error) {
	return scanRule(r.q.QueryRow(ctx, ruleSelect+` WHERE id=$1`, id))
}
