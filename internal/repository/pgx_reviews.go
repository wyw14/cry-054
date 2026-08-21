package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wyw14/cry-054/internal/domain"
)

func (r *pgxRepositories) AppendReview(ctx context.Context, decision domain.ReviewDecision) error {
	metadata, err := json.Marshal(decision.Metadata)
	if err != nil {
		return fmt.Errorf("marshal review metadata: %w", err)
	}
	_, err = r.q.Exec(ctx, `
		INSERT INTO review_actions(id, claim_id, actor_id, action, reason, metadata, created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, decision.ID, decision.ClaimID,
		decision.ActorID, decision.Action, decision.Reason, metadata, decision.CreatedAt)
	return mapPgxError(err)
}

func (r *pgxRepositories) ListReviews(ctx context.Context, claimID string) ([]domain.ReviewDecision, error) {
	rows, err := r.q.Query(ctx, `
		SELECT id, claim_id, actor_id, action, reason, metadata, created_at
		FROM review_actions WHERE claim_id=$1 ORDER BY created_at, id`, claimID)
	if err != nil {
		return nil, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.ReviewDecision, 0)
	for rows.Next() {
		var decision domain.ReviewDecision
		var metadata []byte
		if err := rows.Scan(&decision.ID, &decision.ClaimID, &decision.ActorID, &decision.Action, &decision.Reason, &metadata, &decision.CreatedAt); err != nil {
			return nil, mapPgxError(err)
		}
		if err := json.Unmarshal(metadata, &decision.Metadata); err != nil {
			return nil, fmt.Errorf("decode review metadata: %w", err)
		}
		items = append(items, decision)
	}
	return items, mapPgxError(rows.Err())
}

func (r *pgxRepositories) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	detail, err := json.Marshal(event.Detail)
	if err != nil {
		return fmt.Errorf("marshal audit detail: %w", err)
	}
	_, err = r.q.Exec(ctx, `
		INSERT INTO audit_events(request_id, actor_id, action, subject_type, subject_id, detail, created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, event.RequestID, event.ActorID, event.Action,
		event.SubjectType, event.SubjectID, detail, event.CreatedAt)
	return mapPgxError(err)
}

func (r *pgxRepositories) ListAudit(ctx context.Context, filter domain.AuditFilter, page domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	rows, err := r.q.Query(ctx, `
		SELECT id, request_id, actor_id, action, subject_type, subject_id, detail, created_at,
		       count(*) OVER()
		FROM audit_events
		WHERE ($1='' OR actor_id=$1) AND ($2='' OR subject_type=$2)
		  AND ($3='' OR subject_id=$3) AND ($4='' OR action=$4)
		  AND ($5::timestamptz IS NULL OR created_at >= $5)
		  AND ($6::timestamptz IS NULL OR created_at <= $6)
		ORDER BY created_at DESC, id DESC LIMIT $7 OFFSET $8`,
		filter.ActorID, filter.SubjectType, filter.SubjectID, filter.Action,
		filter.From, filter.To, page.PageSize, page.Offset())
	if err != nil {
		return domain.Page[domain.AuditEvent]{}, mapPgxError(err)
	}
	defer rows.Close()
	items := make([]domain.AuditEvent, 0, page.PageSize)
	total := 0
	for rows.Next() {
		var event domain.AuditEvent
		var detail []byte
		if err := rows.Scan(&event.ID, &event.RequestID, &event.ActorID, &event.Action, &event.SubjectType, &event.SubjectID, &detail, &event.CreatedAt, &total); err != nil {
			return domain.Page[domain.AuditEvent]{}, mapPgxError(err)
		}
		if err := json.Unmarshal(detail, &event.Detail); err != nil {
			return domain.Page[domain.AuditEvent]{}, fmt.Errorf("decode audit detail: %w", err)
		}
		items = append(items, event)
	}
	return domain.Page[domain.AuditEvent]{Items: items, Page: page.Page, PageSize: page.PageSize, Total: total}, mapPgxError(rows.Err())
}
