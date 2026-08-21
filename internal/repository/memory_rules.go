package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/wyw14/cry-054/internal/domain"
)

func (r *memoryRepositories) CreateRule(_ context.Context, rule domain.RuleVersion) error {
	if _, exists := r.data.rules[rule.ID]; exists {
		return fmt.Errorf("rule %s: %w", rule.ID, domain.ErrConflict)
	}
	for _, existing := range r.data.rules {
		if existing.ProjectID == rule.ProjectID && existing.Version == rule.Version {
			return fmt.Errorf("project %s rule version %d: %w", rule.ProjectID, rule.Version, domain.ErrConflict)
		}
	}
	r.data.rules[rule.ID] = rule
	return nil
}

func (s *MemoryStore) CreateRule(ctx context.Context, rule domain.RuleVersion) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.CreateRule(ctx, rule) })
}

func (r *memoryRepositories) PublishRule(_ context.Context, rule domain.RuleVersion) error {
	current, exists := r.data.rules[rule.ID]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Published() {
		return fmt.Errorf("rule %s already published: %w", rule.ID, domain.ErrConflict)
	}
	if !rule.Published() {
		return fmt.Errorf("rule %s missing publication time: %w", rule.ID, domain.ErrInvalidInput)
	}
	r.data.rules[rule.ID] = rule
	return nil
}

func (s *MemoryStore) PublishRule(ctx context.Context, rule domain.RuleVersion) error {
	return s.withWrite(func(r *memoryRepositories) error { return r.PublishRule(ctx, rule) })
}

func (r *memoryRepositories) ListRules(_ context.Context, projectID string) ([]domain.RuleVersion, error) {
	items := make([]domain.RuleVersion, 0)
	for _, rule := range r.data.rules {
		if rule.ProjectID == projectID {
			rule.Segments = append([]domain.RateSegment(nil), rule.Segments...)
			items = append(items, rule)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	return items, nil
}

func (s *MemoryStore) ListRules(ctx context.Context, projectID string) ([]domain.RuleVersion, error) {
	var output []domain.RuleVersion
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.ListRules(ctx, projectID)
		return err
	})
	return output, err
}

func (r *memoryRepositories) GetRule(_ context.Context, id string) (domain.RuleVersion, error) {
	rule, exists := r.data.rules[id]
	if !exists {
		return domain.RuleVersion{}, domain.ErrNotFound
	}
	rule.Segments = append([]domain.RateSegment(nil), rule.Segments...)
	return rule, nil
}

func (s *MemoryStore) GetRule(ctx context.Context, id string) (domain.RuleVersion, error) {
	var output domain.RuleVersion
	err := s.withRead(func(r *memoryRepositories) error {
		var err error
		output, err = r.GetRule(ctx, id)
		return err
	})
	return output, err
}
