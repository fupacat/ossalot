package github

import (
	"context"
	"fmt"
	"net/http"
)

// TRACEABILITY_MAPPING: REQ-015
type ReferrerItem struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
	Uniques  int    `json:"uniques"`
}

// TRACEABILITY_MAPPING: REQ-015
type TrafficSnapshot struct {
	ViewsTotal   int            `json:"views_total"`
	ViewsUnique  int            `json:"views_unique"`
	ClonesTotal  int            `json:"clones_total"`
	ClonesUnique int            `json:"clones_unique"`
	Referrers    []ReferrerItem `json:"referrers"`
}

// TRACEABILITY_MAPPING: REQ-015
func (c *Client) FetchTrafficSnapshot(ctx context.Context, token, owner, repo string) (TrafficSnapshot, error) {
	var viewsResp struct {
		Count   int `json:"count"`
		Uniques int `json:"uniques"`
	}
	viewsURL := fmt.Sprintf("%s/repos/%s/%s/traffic/views", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, viewsURL, token, nil, &viewsResp); err != nil {
		return TrafficSnapshot{}, err
	}

	var clonesResp struct {
		Count   int `json:"count"`
		Uniques int `json:"uniques"`
	}
	clonesURL := fmt.Sprintf("%s/repos/%s/%s/traffic/clones", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, clonesURL, token, nil, &clonesResp); err != nil {
		return TrafficSnapshot{}, err
	}

	var referrers []ReferrerItem
	refURL := fmt.Sprintf("%s/repos/%s/%s/traffic/popular/referrers", c.APIBaseURL, owner, repo)
	if err := c.doJSON(ctx, http.MethodGet, refURL, token, nil, &referrers); err != nil {
		return TrafficSnapshot{}, err
	}

	return TrafficSnapshot{
		ViewsTotal:   viewsResp.Count,
		ViewsUnique:  viewsResp.Uniques,
		ClonesTotal:  clonesResp.Count,
		ClonesUnique: clonesResp.Uniques,
		Referrers:    referrers,
	}, nil
}
