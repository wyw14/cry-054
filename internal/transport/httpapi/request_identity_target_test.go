package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCanceledRequestRetainsIngressRequestIDInErrorContract(t *testing.T) {
	_, _, router := testRouter(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/claims", bytes.NewReader([]byte(`{"claimant_id":"person-1"}`))).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "req-canceled-contract")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if got := response.Header().Get("X-Request-ID"); got != "req-canceled-contract" {
		t.Fatalf("response request id = %q, want ingress id", got)
	}
	if got := body["request_id"]; got != "req-canceled-contract" {
		t.Fatalf("error body request id = %v, want ingress id", got)
	}
}
