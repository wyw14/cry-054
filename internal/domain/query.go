package domain

import (
	"fmt"
	"sort"
	"strings"
)

type PageRequest struct {
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
	SortBy    string   `json:"sort_by"`
	SortOrder string   `json:"sort_order"`
	Statuses  []string `json:"statuses,omitempty"`
}

func (p PageRequest) Normalize(allowedSortFields map[string]bool) (PageRequest, error) {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
	if p.Page < 1 || p.PageSize < 1 || p.PageSize > 100 {
		return PageRequest{}, fmt.Errorf("pagination outside bounds: %w", ErrInvalidInput)
	}
	p.SortBy = strings.TrimSpace(p.SortBy)
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
	if !allowedSortFields[p.SortBy] {
		return PageRequest{}, fmt.Errorf("sort field %q is not allowed: %w", p.SortBy, ErrInvalidInput)
	}
	p.SortOrder = strings.ToLower(strings.TrimSpace(p.SortOrder))
	if p.SortOrder == "" {
		p.SortOrder = "desc"
	}
	if p.SortOrder != "asc" && p.SortOrder != "desc" {
		return PageRequest{}, fmt.Errorf("sort order %q: %w", p.SortOrder, ErrInvalidInput)
	}
	unique := make(map[string]struct{}, len(p.Statuses))
	statuses := make([]string, 0, len(p.Statuses))
	for _, status := range p.Statuses {
		status = strings.TrimSpace(status)
		if status == "" {
			continue
		}
		if _, exists := unique[status]; exists {
			continue
		}
		unique[status] = struct{}{}
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	p.Statuses = statuses
	return p, nil
}

func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type Page[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}
