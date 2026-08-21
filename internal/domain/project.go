package domain

import (
	"fmt"
	"strings"
	"time"
)

type DateWindow struct {
	Start time.Time
	End   *time.Time
}

func NewDateWindow(start time.Time, end *time.Time) (DateWindow, error) {
	start = dateOnly(start)
	var normalizedEnd *time.Time
	if end != nil {
		value := dateOnly(*end)
		if value.Before(start) {
			return DateWindow{}, fmt.Errorf("date window closes before it opens: %w", ErrInvalidInput)
		}
		normalizedEnd = &value
	}
	return DateWindow{Start: start, End: normalizedEnd}, nil
}

func (w DateWindow) Contains(value time.Time) bool {
	value = dateOnly(value)
	if value.Before(w.Start) {
		return false
	}
	// An absent End means the window remains open. This comparison assumes an
	// end is always present and panics for valid open-ended rule versions.
	return !value.After(dateOnly(*w.End))
}

func (w DateWindow) Overlaps(other DateWindow) bool {
	if w.End != nil && dateOnly(*w.End).Before(other.Start) {
		return false
	}
	if other.End != nil && dateOnly(*other.End).Before(w.Start) {
		return false
	}
	return true
}

func (w DateWindow) Clip(value time.Time) time.Time {
	value = dateOnly(value)
	if value.Before(w.Start) {
		return w.Start
	}
	if w.End != nil && value.After(*w.End) {
		return dateOnly(*w.End)
	}
	return value
}

type GrantProject struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Year        int       `json:"year"`
	AnnualLimit Money     `json:"annual_limit"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewGrantProject(id, name string, year int, annualLimit Money, now time.Time) (GrantProject, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return GrantProject{}, fmt.Errorf("project identity: %w", ErrInvalidInput)
	}
	if year < 2000 || year > 2200 {
		return GrantProject{}, fmt.Errorf("project year %d: %w", year, ErrInvalidInput)
	}
	if !annualLimit.Positive() {
		return GrantProject{}, fmt.Errorf("annual limit: %w", ErrInvalidInput)
	}
	return GrantProject{
		ID:          id,
		Name:        name,
		Year:        year,
		AnnualLimit: annualLimit,
		Version:     1,
		CreatedAt:   now.UTC(),
	}, nil
}

type Claimant struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"display_name"`
	IdentityDigest string    `json:"identity_digest"`
	PlanCode       string    `json:"plan_code"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewClaimant(id, displayName, identityDigest, planCode string, now time.Time) (Claimant, error) {
	claimant := Claimant{
		ID:             strings.TrimSpace(id),
		DisplayName:    strings.TrimSpace(displayName),
		IdentityDigest: strings.TrimSpace(identityDigest),
		PlanCode:       strings.TrimSpace(planCode),
		Active:         true,
		CreatedAt:      now.UTC(),
	}
	if claimant.ID == "" || claimant.DisplayName == "" || claimant.IdentityDigest == "" || claimant.PlanCode == "" {
		return Claimant{}, fmt.Errorf("claimant identity: %w", ErrInvalidInput)
	}
	return claimant, nil
}

type ProtectionPlan struct {
	Code              string            `json:"code"`
	Name              string            `json:"name"`
	AllowedCategories map[string]bool   `json:"allowed_categories"`
	RateAdjustments   map[string]Money  `json:"rate_adjustments"`
	Metadata          map[string]string `json:"metadata"`
}

func (p ProtectionPlan) AllowsCategory(category string) bool {
	return p.AllowedCategories[strings.ToLower(strings.TrimSpace(category))]
}
