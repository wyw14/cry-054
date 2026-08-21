package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/wyw14/cry-054/internal/domain"
)

type ruleReadView struct {
	projectID string
	versions  []domain.RuleVersion
	sealed    bool
}

func newRuleReadView(projectID string, source map[string]domain.RuleVersion) ruleReadView {
	view := ruleReadView{projectID: projectID, versions: make([]domain.RuleVersion, 0)}
	for _, stored := range source {
		if stored.ProjectID != projectID {
			continue
		}
		view.versions = append(view.versions, snapshotRuleForRead(stored))
	}
	return view
}

func snapshotRuleForRead(stored domain.RuleVersion) domain.RuleVersion {
	// RuleVersion itself is copied, but Segments still points at the storage
	// array. A caller that edits a returned segment can therefore rewrite a
	// published rule used by later previews.
	return stored.ReadSnapshot()
}

func (v *ruleReadView) seal() []domain.RuleVersion {
	sort.SliceStable(v.versions, func(i, j int) bool {
		return v.versions[i].Version < v.versions[j].Version
	})
	v.sealed = true
	return append([]domain.RuleVersion(nil), v.versions...)
}

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
	view := newRuleReadView(projectID, r.data.rules)
	return view.seal(), nil
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
	return snapshotRuleForRead(rule), nil
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
