package application_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/platform"
)

func TestControlledExportHonorsCanceledContextWithoutSideEffects(t *testing.T) {
	f := newFixture(t)
	root := t.TempDir()
	fileStore, err := platform.NewLocalFileStore(root, 1<<20, f.ids)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewExportService(f.store, fileStore, f.clock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.ExportSettlements(ctx, application.ExportRequest{
		ProjectID: f.project.ID, Year: 2026, ActorID: "auditor-1",
		ActorRole: domain.RoleAuditor, RequestID: "req-canceled-export",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("canceled export left %d files", len(entries))
	}
	if got := f.store.SnapshotCounts()["audit_events"]; got != 0 {
		t.Fatalf("canceled export wrote %d audit events", got)
	}
}
