package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/application"
	"github.com/wyw14/cry-054/internal/domain"
	"github.com/wyw14/cry-054/internal/platform"
	"github.com/wyw14/cry-054/internal/repository"
	"github.com/wyw14/cry-054/internal/transport/httpapi"
	"go.uber.org/zap"
)

func testRouter(t *testing.T) (*httpapi.Services, *repository.MemoryStore, http.Handler) {
	t.Helper()
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	clock := platform.FixedClock{Value: now}
	ids := &platform.SequenceIDGenerator{}
	project, err := domain.NewGrantProject("project-1", "年度补助", 2026, domain.MustMoney("50000.00"), now)
	if err != nil {
		t.Fatal(err)
	}
	claimant, err := domain.NewClaimant("person-1", "申请人", "digest-person", "enhanced", now)
	if err != nil {
		t.Fatal(err)
	}
	published := now.Add(-time.Hour)
	rule := domain.RuleVersion{
		ID: "rule-v1", ProjectID: project.ID, Version: 1,
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Cap:           domain.MustMoney("30000.00"), PublishedAt: &published,
		Segments:   []domain.RateSegment{{Threshold: domain.MustMoney("30000.00"), Rate: decimal.RequireFromString("0.8")}},
		Conditions: domain.RuleConditions{Categories: map[string]bool{"medical": true}, PlanCodes: map[string]bool{"enhanced": true}, MinAmount: domain.MustMoney("1.00")},
	}
	store := repository.NewMemoryStore()
	store.Seed([]domain.GrantProject{project}, []domain.Claimant{claimant}, []domain.RuleVersion{rule})
	fileStore, err := platform.NewLocalFileStore(t.TempDir(), 1<<20, ids)
	if err != nil {
		t.Fatal(err)
	}
	notifier := platform.NewLocalNotifier(clock)
	services := httpapi.Services{
		Claims:         application.NewClaimService(store, clock, ids),
		Previews:       application.NewPreviewService(store, clock),
		Settlements:    application.NewSettlementService(store, clock, ids),
		Corrections:    application.NewCorrectionService(store, clock, ids),
		Reviews:        application.NewReviewService(store, clock, ids, notifier),
		Rules:          application.NewRuleService(store, clock),
		Reconciliation: application.NewReconciliationService(store),
		Exports:        application.NewExportService(store, fileStore, clock),
	}
	router := httpapi.NewRouter(services, httpapi.RouterConfig{
		Logger: zap.NewNop(), AllowedOrigins: []string{"http://localhost:5173"},
		RequestTimeout: time.Second, Ready: func() bool { return true },
	})
	return &services, store, router
}

func TestCreateClaimHTTPRunsHandlerAndPersistsBusinessEffect(t *testing.T) {
	_, store, router := testRouter(t)
	body := []byte(`{"claimant_id":"person-1","project_id":"project-1","category":"medical","receipt_summary":"门诊费用","receipt_digest":"receipt-001","occurred_on":"2026-05-10","amount":"1200.00"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "claim-create-001")
	request.Header.Set("X-Request-ID", "req-http-create")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") != "req-http-create" {
		t.Fatalf("missing stable request id: %q", response.Header().Get("X-Request-ID"))
	}
	if got := store.SnapshotCounts()["claims"]; got != 1 {
		t.Fatalf("got %d claims, want 1", got)
	}
}

func TestCreateClaimHTTPReturnsStableFieldError(t *testing.T) {
	_, _, router := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader([]byte(`{"claimant_id":"person-1"}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "req-http-invalid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "VALIDATION_FAILED" || body["request_id"] != "req-http-invalid" {
		t.Fatalf("unexpected error body: %+v", body)
	}
}

func TestListClaimsRejectsUnapprovedSortField(t *testing.T) {
	_, _, router := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/claims?sort_by=amount%3BDROP+TABLE", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
