package service

import (
	"context"
	"fmt"
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
	batch := newDueBatch(now, s.jobs)
	due := batch.detach()
	s.mu.Unlock()
	errorsFound := make([]error, 0)
	for i, job := range due {
		if err := ctx.Err(); err != nil {
			errorsFound = append(errorsFound, err)
			// Cancellation arrived before this job started. detach() already
			// pulled every due job out of s.jobs, so the jobs from here onward
			// have no home. Restore them to the pending set under the lock so
			// they remain eligible for a later round; jobs already completed
			// (index < i) live in s.completed and are not touched here.
			s.mu.Lock()
			batch.restore(due[i:])
			s.mu.Unlock()
			break
		}
		if err := job.Operation(ctx); err != nil {
			errorsFound = append(errorsFound, fmt.Errorf("job %s: %w", job.ID, err))
			s.mu.Lock()
			batch.retry(job)
			s.jobs[job.Key] = batch.pending[job.Key]
			s.mu.Unlock()
			continue
		}
		s.mu.Lock()
		batch.complete(job.Key)
		s.completed[job.Key] = now.UTC()
		s.mu.Unlock()
	}
	return errorsFound
}
