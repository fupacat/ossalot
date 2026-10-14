package community

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LiveFeedTransport answers the Scanner's /feeds/{platform}?q= contract from public community APIs.
// Scores are fixed per platform (ADR-024), not measured relevance. Lobsters has no search API,
// so the lobsters platform is answered from Ask HN and labeled that way.
// TRACEABILITY_MAPPING: REQ-050
type LiveFeedTransport struct {
	Base             http.RoundTripper
	AlgoliaURL       string
	StackExchangeURL string
	RedditURL        string
	DevToURL         string
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	platform := strings.TrimPrefix(req.URL.Path, "/feeds/")
	keyword := strings.TrimSpace(req.URL.Query().Get("q"))
	if keyword == "" {
		keyword = "open source"
	}
	ctx, cancel := context.WithTimeout(req.Context(), 6*time.Second)
	defer cancel()

	var items []communityFeedItem
	switch platform {
	case "hackernews":
		items = t.algolia(ctx, keyword, "story", 3, "Hacker News")
	case "lobsters":
		items = t.algolia(ctx, keyword, "ask_hn", 2, "Ask HN")
	case "stackoverflow":
		items = t.stackOverflow(ctx, keyword)
	case "reddit":
		items = t.reddit(ctx, keyword)
	case "devto":
		items = t.devTo(ctx, keyword)
	}
	raw, _ := json.Marshal(items)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
		Request:    req,
	}, nil
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) getJSON(ctx context.Context, apiURL string, out any) bool {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	req.Header.Set("User-Agent", "ossalot/1.0 (open-source community scanner)")
	resp, err := t.Base.RoundTrip(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(out) == nil
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) algolia(ctx context.Context, keyword, tag string, limit int, label string) []communityFeedItem {
	var payload struct {
		Hits []struct {
			ObjectID string `json:"objectID"`
			Title    string `json:"title"`
			URL      string `json:"url"`
			Author   string `json:"author"`
			Points   int    `json:"points"`
		} `json:"hits"`
	}
	apiURL := fmt.Sprintf("%s/api/v1/search?query=%s&tags=%s&hitsPerPage=%d", t.AlgoliaURL, url.QueryEscape(keyword), tag, limit)
	if !t.getJSON(ctx, apiURL, &payload) {
		return nil
	}
	out := []communityFeedItem{}
	for _, h := range payload.Hits {
		if h.Title == "" {
			continue
		}
		link := h.URL
		if link == "" {
			link = "https://news.ycombinator.com/item?id=" + h.ObjectID
		}
		score := 82
		if h.Points > 50 {
			score = 91
		}
		out = append(out, communityFeedItem{ID: h.ObjectID, Title: h.Title, URL: link, Author: h.Author, Community: label, Score: score})
	}
	return out
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) stackOverflow(ctx context.Context, keyword string) []communityFeedItem {
	var payload struct {
		Items []struct {
			QuestionID int    `json:"question_id"`
			Title      string `json:"title"`
			Link       string `json:"link"`
			Owner      struct {
				DisplayName string `json:"display_name"`
			} `json:"owner"`
		} `json:"items"`
	}
	apiURL := fmt.Sprintf("%s/2.3/search/advanced?order=desc&sort=relevance&q=%s&site=stackoverflow&pagesize=3", t.StackExchangeURL, url.QueryEscape(keyword))
	if !t.getJSON(ctx, apiURL, &payload) {
		return nil
	}
	out := []communityFeedItem{}
	for _, item := range payload.Items {
		out = append(out, communityFeedItem{ID: fmt.Sprintf("%d", item.QuestionID), Title: item.Title, URL: item.Link, Author: item.Owner.DisplayName, Community: "StackOverflow", Score: 88})
	}
	return out
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) reddit(ctx context.Context, keyword string) []communityFeedItem {
	var payload struct {
		Data struct {
			Children []struct {
				Data struct {
					ID                    string `json:"id"`
					Title                 string `json:"title"`
					Permalink             string `json:"permalink"`
					Author                string `json:"author"`
					SubredditNamePrefixed string `json:"subreddit_name_prefixed"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	apiURL := fmt.Sprintf("%s/search.json?q=%s&sort=relevance&limit=3", t.RedditURL, url.QueryEscape(keyword))
	if !t.getJSON(ctx, apiURL, &payload) {
		return nil
	}
	out := []communityFeedItem{}
	for _, c := range payload.Data.Children {
		if c.Data.Title == "" {
			continue
		}
		community := c.Data.SubredditNamePrefixed
		if community == "" {
			community = "Reddit"
		}
		out = append(out, communityFeedItem{ID: c.Data.ID, Title: c.Data.Title, URL: "https://www.reddit.com" + c.Data.Permalink, Author: c.Data.Author, Community: community, Score: 86})
	}
	return out
}

// TRACEABILITY_MAPPING: REQ-050
func (t *LiveFeedTransport) devTo(ctx context.Context, keyword string) []communityFeedItem {
	var articles []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		URL   string `json:"url"`
		User  struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	// Dev.to filters by a single tag, so the keyword's first word becomes the tag
	tag := strings.ToLower(strings.ReplaceAll(strings.Fields(keyword)[0], "-", ""))
	apiURL := fmt.Sprintf("%s/api/articles?per_page=3&tag=%s", t.DevToURL, url.QueryEscape(tag))
	if !t.getJSON(ctx, apiURL, &articles) {
		return nil
	}
	out := []communityFeedItem{}
	for _, a := range articles {
		out = append(out, communityFeedItem{ID: fmt.Sprintf("%d", a.ID), Title: a.Title, URL: a.URL, Author: a.User.Username, Community: "Dev.to", Score: 84})
	}
	return out
}
