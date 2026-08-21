package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/wyw14/cry-054/internal/domain"
)

func TestMoneyUsesFixedScaleDecimal(t *testing.T) {
	money, err := domain.ParseMoney("100.10")
	if err != nil {
		t.Fatal(err)
	}
	result := money.Mul(decimal.RequireFromString("0.33"))
	if got, want := result.String(), "33.03"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(encoded), `"33.03"`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestMoneyRejectsUnsupportedPrecision(t *testing.T) {
	if _, err := domain.ParseMoney("1.001"); err == nil {
		t.Fatal("expected precision error")
	}
}
