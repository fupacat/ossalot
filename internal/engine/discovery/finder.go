package discovery

import (
	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-023
func ScoreAndFilterCandidates(repo db.Repository, profile db.CapabilityProfile, candidates []github.IssueCandidate) []db.Opportunity {
	panic("stub: REQ-023 not yet implemented")
}
