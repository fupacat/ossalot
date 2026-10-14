package db

import (
	"path/filepath"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-004
func TestRepositoryUpsertKeepsOtherRepositories(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "repos.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	first, err := store.UpsertRepository(Repository{Owner: "octocat", Name: "one", FullName: "octocat/one"})
	if err != nil {
		t.Fatalf("insert first repository failed: %v", err)
	}

	// A new repository without an ID must not replace the existing one, and gets its own ID
	second, err := store.UpsertRepository(Repository{Owner: "octocat", Name: "two", FullName: "octocat/two"})
	if err != nil || second.ID == "" || second.ID == first.ID {
		t.Fatalf("second repository should get its own ID: err=%v repo=%+v", err, second)
	}
	if got := store.ListRepositories(); len(got) != 2 {
		t.Fatalf("expected both repositories to be kept, got %d", len(got))
	}

	// A caller-supplied ID that matches nothing is kept as given
	explicit, err := store.UpsertRepository(Repository{ID: "repo-explicit", Owner: "octocat", Name: "three", FullName: "octocat/three"})
	if err != nil || explicit.ID != "repo-explicit" {
		t.Fatalf("explicit ID should be kept: err=%v repo=%+v", err, explicit)
	}

	// Saving a repository with a new audit attached keeps that audit
	updated, err := store.UpsertRepository(Repository{
		ID:          first.ID,
		Owner:       "octocat",
		Name:        "one",
		FullName:    "octocat/one",
		LatestAudit: &AuditRecord{ID: "audit-2", OverallScore: 90},
	})
	if err != nil || updated.LatestAudit == nil || updated.LatestAudit.ID != "audit-2" {
		t.Fatalf("upsert should keep the audit it was given: err=%v repo=%+v", err, updated)
	}

	if _, ok := store.GetRepository("repo-missing"); ok {
		t.Fatalf("GetRepository should report false for an unknown ID")
	}

	// Deleting an unknown ID removes nothing
	if store.DeleteRepository("repo-missing") {
		t.Fatalf("DeleteRepository should report false for an unknown ID")
	}
	if got := store.ListRepositories(); len(got) != 3 {
		t.Fatalf("deleting an unknown ID must keep all repositories, got %d", len(got))
	}
}

// TRACEABILITY_MAPPING: REQ-014
func TestMetricSnapshotsStayWithTheirRepository(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	a, err := store.UpsertRepository(Repository{Owner: "octocat", Name: "a", FullName: "octocat/a"})
	if err != nil {
		t.Fatalf("insert repository a failed: %v", err)
	}
	b, err := store.UpsertRepository(Repository{Owner: "octocat", Name: "b", FullName: "octocat/b"})
	if err != nil {
		t.Fatalf("insert repository b failed: %v", err)
	}

	// b is not the first repository, so the snapshot is matched past a
	if err := store.UpsertMetricSnapshot(MetricSnapshot{RepoID: b.ID, SnapshotDate: "2026-10-12"}); err != nil {
		t.Fatalf("snapshot for b failed: %v", err)
	}
	if err := store.UpsertMetricSnapshot(MetricSnapshot{RepoID: a.ID, SnapshotDate: "2026-10-12"}); err != nil {
		t.Fatalf("snapshot for a failed: %v", err)
	}

	gotB := store.ListMetricSnapshots(b.ID)
	if len(gotB) != 1 || gotB[0].RepoID != b.ID {
		t.Fatalf("listing b should return only b's snapshot, got %+v", gotB)
	}
}

// TRACEABILITY_MAPPING: REQ-020
func TestOpportunityRecordsStayDistinct(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "opps.db"))
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}

	// Profiles for two repositories are stored side by side
	if err := store.SaveCapabilityProfile(CapabilityProfile{RepoID: "repo-1", Summary: "first"}); err != nil {
		t.Fatalf("save profile for repo-1 failed: %v", err)
	}
	if err := store.SaveCapabilityProfile(CapabilityProfile{RepoID: "repo-2", Summary: "second"}); err != nil {
		t.Fatalf("save profile for repo-2 failed: %v", err)
	}
	if p, ok := store.GetCapabilityProfile("repo-2"); !ok || p.Summary != "second" {
		t.Fatalf("expected repo-2's profile, got %+v (found=%v)", p, ok)
	}

	// Opportunities with different external IDs are separate records
	o1, err := store.UpsertOpportunity(Opportunity{RepoID: "repo-1", ExternalID: "ext-1", Title: "first"})
	if err != nil {
		t.Fatalf("upsert ext-1 failed: %v", err)
	}
	o2, err := store.UpsertOpportunity(Opportunity{RepoID: "repo-2", ExternalID: "ext-2", Title: "second"})
	if err != nil {
		t.Fatalf("upsert ext-2 failed: %v", err)
	}
	if o1.ID == o2.ID {
		t.Fatalf("distinct opportunities must get distinct IDs, both got %s", o1.ID)
	}

	// Listing one repository leaves out the other repository's records
	list := store.ListOpportunities("repo-2")
	if len(list) != 1 || list[0].ExternalID != "ext-2" {
		t.Fatalf("listing repo-2 should return only ext-2, got %+v", list)
	}
}
