package profiler

import (
	"testing"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-021
func TestBuildCapabilityProfile(t *testing.T) {
	// 1. Go repo with explicit topics and FEATURE section in README
	goProf := BuildCapabilityProfile(db.Repository{
		ID:              "repo-1",
		Name:            "ossalot",
		FullName:        "eric/ossalot",
		Description:     "OSS standards and discovery platform",
		PrimaryLanguage: "Go",
		Topics:          []string{"OpenSSF", "Scorecard"},
	}, "# OSSAlot\n## Features\nAutomated scorecard checks")

	if len(goProf.ProblemStatements) != 3 || len(goProf.SearchQueries) != 2 || len(goProf.EcosystemKeywords) != 2 {
		t.Fatalf("unexpected Go profile: %+v", goProf)
	}

	// 2. Python repo without topics or description
	pyProf := BuildCapabilityProfile(db.Repository{
		ID:              "repo-2",
		Name:            "pytool",
		FullName:        "eric/pytool",
		PrimaryLanguage: "Python",
	}, "Minimal readme")
	if len(pyProf.EcosystemKeywords) != 2 || len(pyProf.RecommendedCommunities) == 0 {
		t.Fatalf("unexpected Python profile: %+v", pyProf)
	}

	// 3. Repo with empty language
	genericProf := BuildCapabilityProfile(db.Repository{
		ID:       "repo-3",
		Name:     "anytool",
		FullName: "eric/anytool",
	}, "")
	if genericProf.RepoID != "repo-3" || len(genericProf.RecommendedCommunities) == 0 {
		t.Fatalf("unexpected generic profile: %+v", genericProf)
	}
}
