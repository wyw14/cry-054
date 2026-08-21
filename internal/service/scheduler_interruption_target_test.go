package service

import (
	"context"
	"testing"
	"time"
)

func TestInterruptedDueBatchRetainsJobsThatNeverStarted(t *testing.T) {
	scheduler := NewLocalScheduler()
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	ran := make([]string, 0, 2)
	jobs := []ScheduledJob{
		{ID: "job-a", Key: "a", RunAt: now, Operation: func(context.Context) error { ran = append(ran, "a"); cancel(); return nil }},
		{ID: "job-b", Key: "b", RunAt: now, Operation: func(context.Context) error { ran = append(ran, "b"); return nil }},
	}
	for _, job := range jobs {
		if err := scheduler.Schedule(job); err != nil {
			t.Fatal(err)
		}
	}
	if errorsFound := scheduler.RunDue(ctx, now); len(errorsFound) != 1 {
		t.Fatalf("first run errors = %v, want cancellation", errorsFound)
	}
	if errorsFound := scheduler.RunDue(context.Background(), now); len(errorsFound) != 0 {
		t.Fatalf("resume errors = %v", errorsFound)
	}
	if len(ran) != 2 || ran[0] != "a" || ran[1] != "b" {
		t.Fatalf("execution order after resume = %v, want [a b]", ran)
	}
}
