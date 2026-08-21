package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDuplicateReceiptPreservesConflictContract(t *testing.T) {
	_, store, router := testRouter(t)
	body := []byte(`{"claimant_id":"person-1","project_id":"project-1","category":"medical","receipt_summary":"门诊凭证","receipt_digest":"receipt-same","occurred_on":"2026-05-10","amount":"1200.00"}`)

	first := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader(body))
	first.Header.Set("Content-Type", "application/json")
	first.Header.Set("Idempotency-Key", "first-submission")
	firstResult := httptest.NewRecorder()
	router.ServeHTTP(firstResult, first)
	if firstResult.Code != http.StatusCreated {
		t.Fatalf("first submission status=%d body=%s", firstResult.Code, firstResult.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader(body))
	second.Header.Set("Content-Type", "application/json")
	second.Header.Set("Idempotency-Key", "different-request-for-same-receipt")
	second.Header.Set("X-Request-ID", "req-duplicate-receipt")
	secondResult := httptest.NewRecorder()
	router.ServeHTTP(secondResult, second)

	if secondResult.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", secondResult.Code, secondResult.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(secondResult.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["code"] != "DUPLICATE_CLAIM" {
		t.Fatalf("code=%v, want DUPLICATE_CLAIM", response["code"])
	}
	if response["request_id"] != "req-duplicate-receipt" {
		t.Fatalf("request_id=%v", response["request_id"])
	}
	if got := store.SnapshotCounts()["claims"]; got != 1 {
		t.Fatalf("stored claims=%d, want 1", got)
	}
}
