package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// TRACEABILITY_MAPPING: REQ-029
type JobFunc func(ctx context.Context) error

// TRACEABILITY_MAPPING: REQ-029
type ScanJobRecord struct {
	ID         string `json:"id"`
	JobName    string `json:"job_name"`
	Status     string `json:"status"`
	Details    string `json:"details"`
	ExecutedAt string `json:"executed_at"`
}

// TRACEABILITY_MAPPING: REQ-029
type Runner struct {
	mu      sync.Mutex
	jobs    map[string]JobFunc
	history []ScanJobRecord
	nextSeq int
}

// TRACEABILITY_MAPPING: REQ-029
func NewRunner() *Runner {
	return &Runner{
		jobs:    map[string]JobFunc{},
		history: []ScanJobRecord{},
		nextSeq: 1,
	}
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) RegisterJob(name string, fn JobFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[name] = fn
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) TriggerJob(ctx context.Context, name string) (ScanJobRecord, error) {
	r.mu.Lock()
	fn, ok := r.jobs[name]
	r.mu.Unlock()
	if !ok {
		return ScanJobRecord{}, errors.New("job not registered: " + name)
	}

	err := fn(ctx)
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := ScanJobRecord{
		ID:         fmt.Sprintf("job-%d", r.nextSeq),
		JobName:    name,
		Status:     "completed",
		Details:    "ok",
		ExecutedAt: time.Now().UTC().Format(time.RFC3339),
	}
	r.nextSeq++
	if err != nil {
		rec.Status = "failed"
		rec.Details = err.Error()
	}
	r.history = append(r.history, rec)
	return rec, err
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) ListJobs() []ScanJobRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ScanJobRecord, len(r.history))
	copy(out, r.history)
	return out
}
