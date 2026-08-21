package domain_test

import (
	"testing"

	"github.com/wyw14/cry-054/internal/domain"
)

func TestPageRequestAllowsOnlyWhitelistedSortAndNormalizesFilters(t *testing.T) {
	request, err := (domain.PageRequest{Page: 1, PageSize: 20, SortBy: "created_at", SortOrder: "DESC", Statuses: []string{"approved", "approved", "submitted"}}).Normalize(map[string]bool{"created_at": true})
	if err != nil {
		t.Fatal(err)
	}
	if request.SortOrder != "desc" || len(request.Statuses) != 2 {
		t.Fatalf("unexpected normalization: %+v", request)
	}
	if _, err := (domain.PageRequest{SortBy: "created_at; DROP TABLE claims"}).Normalize(map[string]bool{"created_at": true}); err == nil {
		t.Fatal("expected unsafe sort to fail")
	}
}
