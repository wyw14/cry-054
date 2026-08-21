package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wyw14/cry-054/internal/domain"
)

type ExportRequest struct {
	ProjectID string
	Year      int
	ActorID   string
	ActorRole domain.ActorRole
	RequestID string
	Columns   []string
}

type ExportService struct {
	repositories Repositories
	sink         ExportSink
	clock        Clock
}

func NewExportService(repositories Repositories, sink ExportSink, clock Clock) *ExportService {
	return &ExportService{repositories: repositories, sink: sink, clock: clock}
}

func (s *ExportService) ExportSettlements(ctx context.Context, request ExportRequest) (StoredAttachment, error) {
	if request.ActorRole != domain.RoleManager && request.ActorRole != domain.RoleAuditor {
		return StoredAttachment{}, fmt.Errorf("role %s cannot export settlements: %w", request.ActorRole, domain.ErrForbidden)
	}
	allowedColumns := map[string]bool{
		"settlement_id": true,
		"claim_id":      true,
		"rule_version":  true,
		"amount":        true,
		"status":        true,
		"confirmed_at":  true,
	}
	columns := append([]string(nil), request.Columns...)
	if len(columns) == 0 {
		columns = []string{"settlement_id", "claim_id", "rule_version", "amount", "status", "confirmed_at"}
	}
	seen := make(map[string]bool, len(columns))
	for _, column := range columns {
		if !allowedColumns[column] || seen[column] {
			return StoredAttachment{}, domain.ValidationError(domain.FieldViolation{Field: "columns", Message: "contains unknown or duplicate column"})
		}
		seen[column] = true
	}
	settlements, err := s.repositories.ListSettlements(ctx, request.ProjectID, request.Year)
	if err != nil {
		return StoredAttachment{}, fmt.Errorf("list settlements for export: %w", err)
	}
	sort.SliceStable(settlements, func(i, j int) bool {
		if settlements[i].ConfirmedAt.Equal(settlements[j].ConfirmedAt) {
			return settlements[i].ID < settlements[j].ID
		}
		return settlements[i].ConfirmedAt.Before(settlements[j].ConfirmedAt)
	})
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(columns); err != nil {
		return StoredAttachment{}, err
	}
	for _, settlement := range settlements {
		values := map[string]string{
			"settlement_id": settlement.ID,
			"claim_id":      settlement.ClaimID,
			"rule_version":  settlement.RuleVersionID,
			"amount":        settlement.ApprovedAmount.String(),
			"status":        string(settlement.Status),
			"confirmed_at":  settlement.ConfirmedAt.Format(time.RFC3339),
		}
		row := make([]string, len(columns))
		for index, column := range columns {
			row[index] = strings.TrimSpace(values[column])
		}
		if err := writer.Write(row); err != nil {
			return StoredAttachment{}, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return StoredAttachment{}, err
	}
	name := fmt.Sprintf("settlements-%s-%d-%s.csv", request.ProjectID, request.Year, s.clock.Now().UTC().Format("20060102T150405Z"))
	attachment, err := s.sink.Write(ctx, name, "text/csv; charset=utf-8", bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return StoredAttachment{}, fmt.Errorf("store controlled export: %w", err)
	}
	audit, err := domain.NewAuditEvent(request.RequestID, request.ActorID, "settlement.export", "project", request.ProjectID, map[string]any{
		"year":    request.Year,
		"columns": columns,
		"rows":    len(settlements),
		"file_id": attachment.ID,
	}, s.clock.Now())
	if err != nil {
		return StoredAttachment{}, err
	}
	if err := s.repositories.AppendAudit(ctx, audit); err != nil {
		return StoredAttachment{}, fmt.Errorf("append export audit: %w", err)
	}
	return attachment, nil
}
