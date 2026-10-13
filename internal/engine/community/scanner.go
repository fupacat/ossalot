package community

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-026
type Scanner struct {
	BaseURL    string
	HTTPClient *http.Client
}

// TRACEABILITY_MAPPING: REQ-026
func NewScanner(baseURL string, httpClient *http.Client) *Scanner {
	if baseURL == "" {
		baseURL = "https://hn.algolia.com"
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Scanner{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: httpClient,
	}
}

// TRACEABILITY_MAPPING: REQ-026
type communityFeedItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Author    string `json:"author"`
	Community string `json:"community"`
	Score     int    `json:"score"`
}

// TRACEABILITY_MAPPING: REQ-026
var communityPlatforms = []string{
	"hackernews",
	"reddit",
	"stackoverflow",
	"devto",
	"lobsters",
}

// TRACEABILITY_MAPPING: REQ-026
func (s *Scanner) ScanCommunities(ctx context.Context, repo db.Repository, profile db.CapabilityProfile) ([]db.Opportunity, error) {
	keyword := repo.Name
	if len(profile.EcosystemKeywords) > 0 && profile.EcosystemKeywords[0] != "" {
		keyword = profile.EcosystemKeywords[0]
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out := []db.Opportunity{}

	for _, platform := range communityPlatforms {
		endpoint := fmt.Sprintf("%s/feeds/%s?q=%s", s.BaseURL, platform, url.QueryEscape(keyword))
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		resp, err := s.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("community feed %s returned status %d", platform, resp.StatusCode)
		}
		var items []communityFeedItem
		_ = json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()

		for _, item := range items {
			relScore := item.Score
			if relScore <= 0 {
				relScore = 80
			}
			action := "comment_solution"
			if platform == "stackoverflow" {
				action = "answer_question"
			}
			out = append(out, db.Opportunity{
				RepoID:              repo.ID,
				Category:            platform,
				ExternalID:          fmt.Sprintf("%s:%s", platform, item.ID),
				Title:               item.Title,
				URL:                 item.URL,
				SourceAuthor:        item.Author,
				SourceCommunity:     item.Community,
				RelevanceScore:      relScore,
				LegitimacyReason:    fmt.Sprintf("Active %s thread discussing %s where %s directly addresses the question.", platform, keyword, repo.FullName),
				RecommendedAction:   action,
				DraftTitle:          fmt.Sprintf("Re: %s", item.Title),
				DraftBody:           fmt.Sprintf("We ran into this exact %s challenge and open-sourced `%s` (%s) to solve it. Happy to share details if helpful!", keyword, repo.FullName, repo.HTMLURL),
				CommunityRulesNotes: "Disclose maintainer affiliation and lead with a concrete technical answer before linking.",
				Status:              "new",
				DiscoveredAt:        now,
			})
		}
	}
	return out, nil
}

// TRACEABILITY_MAPPING: REQ-027
func GenerateMarketingPlaybooks(repo db.Repository, profile db.CapabilityProfile) []db.Opportunity {
	lang := repo.PrimaryLanguage
	if lang == "" {
		lang = "Open Source"
	}
	summary := profile.Summary
	if summary == "" {
		summary = repo.Description
	}
	now := time.Now().UTC().Format(time.RFC3339)

	return []db.Opportunity{
		{
			RepoID:              repo.ID,
			Category:            "marketing_playbook",
			ExternalID:          fmt.Sprintf("playbook:%s:show-hn", repo.ID),
			Title:               fmt.Sprintf("Show HN Launch Playbook for %s", repo.FullName),
			URL:                 "https://news.ycombinator.com/submit",
			SourceAuthor:        repo.Owner,
			SourceCommunity:     "Hacker News (Show HN)",
			RelevanceScore:      95,
			LegitimacyReason:    "Direct technical Show HN launch sharing architecture and motivation.",
			RecommendedAction:   "publish_post",
			DraftTitle:          fmt.Sprintf("Show HN: %s – %s", repo.Name, summary),
			DraftBody:           fmt.Sprintf("Hi HN, I built %s (%s) in %s to solve %s. Try it out and let me know your feedback.", repo.FullName, repo.HTMLURL, lang, summary),
			CommunityRulesNotes: "Use factual title without marketing superlatives; post a technical top-level comment explaining trade-offs and limitations.",
			Status:              "new",
			DiscoveredAt:        now,
		},
		{
			RepoID:              repo.ID,
			Category:            "marketing_playbook",
			ExternalID:          fmt.Sprintf("playbook:%s:reddit", repo.ID),
			Title:               fmt.Sprintf("Reddit Technical Deep-Dive for %s", repo.FullName),
			URL:                 "https://www.reddit.com/submit",
			SourceAuthor:        repo.Owner,
			SourceCommunity:     fmt.Sprintf("r/%s", strings.ToLower(lang)),
			RelevanceScore:      90,
			LegitimacyReason:    "Architectural walkthrough tailored for ecosystem developers.",
			RecommendedAction:   "publish_post",
			DraftTitle:          fmt.Sprintf("How we built %s in %s: architecture & lessons learned", repo.Name, lang),
			DraftBody:           fmt.Sprintf("We open-sourced %s (%s). Here is how the internals work and when to use it.", repo.FullName, repo.HTMLURL),
			CommunityRulesNotes: "Follow subreddit 90/10 self-promotion ratio; include full architectural content in the self-post body so readers do not have to click out.",
			Status:              "new",
			DiscoveredAt:        now,
		},
		{
			RepoID:              repo.ID,
			Category:            "marketing_playbook",
			ExternalID:          fmt.Sprintf("playbook:%s:devto", repo.ID),
			Title:               fmt.Sprintf("Dev.to Tutorial Article for %s", repo.FullName),
			URL:                 "https://dev.to/new",
			SourceAuthor:        repo.Owner,
			SourceCommunity:     "Dev.to",
			RelevanceScore:      86,
			LegitimacyReason:    "Step-by-step tutorial solving a concrete developer workflow.",
			RecommendedAction:   "publish_post",
			DraftTitle:          fmt.Sprintf("Solving %s in %s with %s", summary, lang, repo.Name),
			DraftBody:           fmt.Sprintf("## Problem\n%s\n\n## Solution with %s\nSee %s for runnable examples.", summary, repo.FullName, repo.HTMLURL),
			CommunityRulesNotes: "Include runnable code snippets and tag with ecosystem tags.",
			Status:              "new",
			DiscoveredAt:        now,
		},
		{
			RepoID:              repo.ID,
			Category:            "marketing_playbook",
			ExternalID:          fmt.Sprintf("playbook:%s:stackoverflow", repo.ID),
			Title:               fmt.Sprintf("StackOverflow Code-First Answer Guide for %s", repo.FullName),
			URL:                 "https://stackoverflow.com/questions",
			SourceAuthor:        repo.Owner,
			SourceCommunity:     fmt.Sprintf("StackOverflow [%s]", strings.ToLower(lang)),
			RelevanceScore:      85,
			LegitimacyReason:    "Self-contained code answer for unanswered community questions.",
			RecommendedAction:   "answer_question",
			DraftTitle:          fmt.Sprintf("Reference Answer Template (%s)", repo.Name),
			DraftBody:           fmt.Sprintf("You can solve this directly in standard %s, or using `%s` (%s) which I maintain.", lang, repo.FullName, repo.HTMLURL),
			CommunityRulesNotes: "Mandatory: provide a complete standalone code solution in the answer body and explicitly state 'Disclaimer: I am the author of this library'.",
			Status:              "new",
			DiscoveredAt:        now,
		},
	}
}
