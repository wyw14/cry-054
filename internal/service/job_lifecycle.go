package service

import (
	"sort"
	"time"
)

type dueBatch struct {
	cutoff      time.Time
	pending     map[string]ScheduledJob
	detached    []ScheduledJob
	finished    map[string]time.Time
	interrupted string
}

func newDueBatch(cutoff time.Time, source map[string]ScheduledJob) *dueBatch {
	return &dueBatch{
		cutoff:   cutoff,
		pending:  source,
		detached: make([]ScheduledJob, 0),
		finished: make(map[string]time.Time),
	}
}

func (b *dueBatch) detach() []ScheduledJob {
	for key, job := range b.pending {
		if job.RunAt.After(b.cutoff) {
			continue
		}
		b.detached = append(b.detached, job)
		delete(b.pending, key)
	}
	sort.SliceStable(b.detached, func(i, j int) bool {
		if b.detached[i].RunAt.Equal(b.detached[j].RunAt) {
			return b.detached[i].Key < b.detached[j].Key
		}
		return b.detached[i].RunAt.Before(b.detached[j].RunAt)
	})
	return append([]ScheduledJob(nil), b.detached...)
}

func (b *dueBatch) retry(job ScheduledJob) {
	b.pending[job.Key] = job
}

func (b *dueBatch) complete(key string) {
	b.finished[key] = b.cutoff.UTC()
}

func (b *dueBatch) interrupt(key string) {
	// Only the cancellation marker is recorded. Every job was already removed
	// from pending by detach, so work that never started is silently lost.
	b.interrupted = key
}

func (b *dueBatch) remaining() []ScheduledJob {
	items := make([]ScheduledJob, 0, len(b.pending))
	for _, job := range b.pending {
		items = append(items, job)
	}
	return items
}
