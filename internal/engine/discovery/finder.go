package discovery

import (
	"fmt"
	"strings"
	"time"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
)

// TRACEABILITY_MAPPING: REQ-023
func ScoreAndFilterCandidates(repo db.Repository, profile db.CapabilityProfile, candidates []github.IssueCandidate) []db.Opportunity {
	out := []db.Opportunity{}
	now := time.Now().UTC().Format(time.RFC3339)

	for _, cand := range candidates {
		if strings.EqualFold(cand.TargetRepoFullName, repo.FullName) {
			continue
		}
		combined := strings.ToLower(cand.Title + " " + cand.Body)
		score := 45
		matchedKeywords := []string{}
		for _, kw := range profile.EcosystemKeywords {
			if kw != "" && strings.Contains(combined, strings.ToLower(kw)) {
				score += 20
				matchedKeywords = append(matchedKeywords, kw)
			}
		}
		for _, lbl := range cand.Labels {
			lowerLbl := strings.ToLower(lbl)
			if strings.Contains(lowerLbl, "help wanted") || strings.Contains(lowerLbl, "enhancement") {
				score += 15
			}
		}
		if score > 100 {
			score = 100
		}
		if score < 70 {
			continue
		}

		kwSummary := strings.Join(matchedKeywords, ", ")
		out = append(out, db.Opportunity{
			RepoID:               repo.ID,
			Category:             "github_issue",
			ExternalID:           fmt.Sprintf("gh:%s#%d", cand.TargetRepoFullName, cand.IssueNumber),
			Title:                cand.Title,
			URL:                  cand.HTMLURL,
			SourceAuthor:         cand.Author,
			SourceCommunity:      cand.TargetRepoFullName,
			RelevanceScore:       score,
			LegitimacyReason:     fmt.Sprintf("Issue in %s directly requests capabilities matching %s (%s).", cand.TargetRepoFullName, repo.FullName, kwSummary),
			RecommendedAction:    "open_pr",
			DraftTitle:           fmt.Sprintf("feat: integrate %s to resolve #%d", repo.Name, cand.IssueNumber),
			DraftBody:            fmt.Sprintf("Hi @%s, `%s` provides native support for %s and can resolve #%d cleanly.", cand.Author, repo.FullName, kwSummary, cand.IssueNumber),
			PRImplementationPlan: fmt.Sprintf("1. Add `%s` dependency to `%s`.\n2. Wire capability (%s) into the module handling issue #%d.\n3. Add regression test verifying #%d.", repo.FullName, cand.TargetRepoFullName, kwSummary, cand.IssueNumber, cand.IssueNumber),
			Status:               "new",
			DiscoveredAt:         now,
		})
	}
	return out
}
