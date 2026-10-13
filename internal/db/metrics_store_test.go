package db

import (
	"path/filepath"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-014
func TestUpsertAndListMetricSnapshots(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	// Nonexistent repo returns error
	if err := store.UpsertMetricSnapshot(MetricSnapshot{RepoID: "missing", SnapshotDate: "2026-10-12"}); err == nil {
		t.Fatalf("expected error for missing repository")
	}

	repo, _ := store.UpsertRepository(Repository{
		Owner:    "octocat",
		Name:     "hello-world",
		FullName: "octocat/hello-world",
	})

	// Empty list before any snapshots exist
	if list := store.ListMetricSnapshots(repo.ID); len(list) != 0 {
		t.Fatalf("expected 0 snapshots initially, got %d", len(list))
	}

	// Insert out-of-order dates and then update an existing date
	if err := store.UpsertMetricSnapshot(MetricSnapshot{
		RepoID:        repo.ID,
		SnapshotDate:  "2026-10-12",
		Stars:         110,
		ClonesTotal:   20,
		AdoptionScore: 70,
	}); err != nil {
		t.Fatalf("UpsertMetricSnapshot 2026-10-12 failed: %v", err)
	}
	if err := store.UpsertMetricSnapshot(MetricSnapshot{
		RepoID:        repo.ID,
		SnapshotDate:  "2026-10-11",
		Stars:         100,
		ClonesTotal:   15,
		AdoptionScore: 65,
	}); err != nil {
		t.Fatalf("UpsertMetricSnapshot 2026-10-11 failed: %v", err)
	}
	// Duplicate date update
	if err := store.UpsertMetricSnapshot(MetricSnapshot{
		RepoID:        repo.ID,
		SnapshotDate:  "2026-10-12",
		Stars:         115,
		ClonesTotal:   25,
		AdoptionScore: 75,
	}); err != nil {
		t.Fatalf("UpsertMetricSnapshot update failed: %v", err)
	}

	snaps := store.ListMetricSnapshots(repo.ID)
	if len(snaps) != 2 {
		t.Fatalf("expected 2 deduplicated snapshots, got %d", len(snaps))
	}
	if snaps[0].SnapshotDate != "2026-10-11" || snaps[1].SnapshotDate != "2026-10-12" || snaps[1].ClonesTotal != 25 {
		t.Fatalf("unexpected snapshots order or values: %+v", snaps)
	}
}
