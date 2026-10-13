package profiler

import (
	"fmt"
	"strings"
	"time"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-021
func BuildCapabilityProfile(repo db.Repository, readmeContent string) db.CapabilityProfile {
	lang := repo.PrimaryLanguage
	if lang == "" {
		lang = "Open Source"
	}
	desc := repo.Description
	if desc == "" {
		desc = fmt.Sprintf("%s toolkit for %s projects", repo.Name, lang)
	}

	keywords := make([]string, 0, len(repo.Topics)+2)
	for _, t := range repo.Topics {
		keywords = append(keywords, strings.ToLower(t))
	}
	if len(keywords) == 0 {
		keywords = append(keywords, strings.ToLower(repo.Name), strings.ToLower(lang))
	}

	problems := []string{
		fmt.Sprintf("Automated %s integration for %s", keywords[0], lang),
		desc,
	}
	if strings.Contains(strings.ToUpper(readmeContent), "FEATURE") {
		problems = append(problems, fmt.Sprintf("Drop-in feature support provided by %s", repo.FullName))
	}

	queries := []string{
		fmt.Sprintf("is:issue is:open %s language:%s", keywords[0], lang),
		fmt.Sprintf("is:issue is:open \"%s\" label:\"help wanted\"", repo.Name),
	}

	communities := []string{"Hacker News (Show HN)", "Dev.to", "Lobsters"}
	switch strings.ToLower(lang) {
	case "go":
		communities = append(communities, "r/golang", "StackOverflow [go]")
	case "python":
		communities = append(communities, "r/Python", "StackOverflow [python]")
	default:
		communities = append(communities, "r/opensource", "StackOverflow")
	}

	return db.CapabilityProfile{
		RepoID:                 repo.ID,
		Summary:                fmt.Sprintf("%s (%s): %s", repo.FullName, lang, desc),
		ProblemStatements:      problems,
		TargetPersonas:         []string{fmt.Sprintf("%s maintainers", lang), "Open-source contributors"},
		EcosystemKeywords:      keywords,
		SearchQueries:          queries,
		RecommendedCommunities: communities,
		UpdatedAt:              time.Now().UTC().Format(time.RFC3339),
	}
}
