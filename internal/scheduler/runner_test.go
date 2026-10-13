package scheduler

import (
	"context"
	"errors"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-029
func TestRunnerRegisterTriggerAndList(t *testing.T) {
	runner := NewRunner()
	ctx := context.Background()

	runner.RegisterJob("metrics_sync", func(ctx context.Context) error {
		return nil
	})
	runner.RegisterJob("community_scan", func(ctx context.Context) error {
		return errors.New("rate limit exceeded")
	})

	if _, err := runner.TriggerJob(ctx, "unknown_job"); err == nil {
		t.Fatalf("expected error triggering unregistered job")
	}

	rec1, err := runner.TriggerJob(ctx, "metrics_sync")
	if err != nil || rec1.Status != "completed" {
		t.Fatalf("expected completed job record, got %+v err=%v", rec1, err)
	}

	rec2, err := runner.TriggerJob(ctx, "community_scan")
	if err == nil || rec2.Status != "failed" || rec2.Details != "rate limit exceeded" {
		t.Fatalf("expected failed job record, got %+v err=%v", rec2, err)
	}

	jobs := runner.ListJobs()
	if len(jobs) != 2 || jobs[0].Status != "completed" || jobs[1].Status != "failed" {
		t.Fatalf("unexpected job history: %+v", jobs)
	}
}
