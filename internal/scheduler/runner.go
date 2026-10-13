package scheduler

import (
	"context"
	"sync"
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
	panic("stub: REQ-029 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) RegisterJob(name string, fn JobFunc) {
	panic("stub: REQ-029 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) TriggerJob(ctx context.Context, name string) (ScanJobRecord, error) {
	panic("stub: REQ-029 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-029
func (r *Runner) ListJobs() []ScanJobRecord {
	panic("stub: REQ-029 not yet implemented")
}
