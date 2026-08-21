package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type ScheduledJob struct {
	ID        string
	RunAt     time.Time
	Key       string
	Operation func(context.Context) error
}

type LocalScheduler struct {
	mu        sync.Mutex
	jobs      map[string]ScheduledJob
	completed map[string]time.Time
}

func NewLocalScheduler() *LocalScheduler {
	return &LocalScheduler{jobs: make(map[string]ScheduledJob), completed: make(map[string]time.Time)}
}

func (s *LocalScheduler) Schedule(job ScheduledJob) error {
	if job.ID == "" || job.Key == "" || job.Operation == nil {
		return fmt.Errorf("scheduled job id, key and operation are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, done := s.completed[job.Key]; done {
		return nil
	}
	if existing, exists := s.jobs[job.Key]; exists {
		if existing.ID != job.ID || !existing.RunAt.Equal(job.RunAt) {
			return fmt.Errorf("job key %s conflicts with existing schedule", job.Key)
		}
		return nil
	}
	s.jobs[job.Key] = job
	return nil
}

func (s *LocalScheduler) RunDue(ctx context.Context, now time.Time) []error {
	s.mu.Lock()
	due := make([]ScheduledJob, 0)
	for key, job := range s.jobs {
		if !job.RunAt.After(now) {
			due = append(due, job)
			delete(s.jobs, key)
		}
	}
	s.mu.Unlock()
	sort.SliceStable(due, func(i, j int) bool {
		if due[i].RunAt.Equal(due[j].RunAt) {
			return due[i].Key < due[j].Key
		}
		return due[i].RunAt.Before(due[j].RunAt)
	})
	errorsFound := make([]error, 0)
	for _, job := range due {
		if err := ctx.Err(); err != nil {
			errorsFound = append(errorsFound, err)
			break
		}
		if err := job.Operation(ctx); err != nil {
			errorsFound = append(errorsFound, fmt.Errorf("job %s: %w", job.ID, err))
			s.mu.Lock()
			s.jobs[job.Key] = job
			s.mu.Unlock()
			continue
		}
		s.mu.Lock()
		s.completed[job.Key] = now.UTC()
		s.mu.Unlock()
	}
	return errorsFound
}
