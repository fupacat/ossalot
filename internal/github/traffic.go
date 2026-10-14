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

// fetchTraffic leaves out at its zero value when GitHub refuses traffic data (403 or 404),
// which it does for repositories the token cannot push to.
// TRACEABILITY_MAPPING: REQ-047
func (c *Client) fetchTraffic(ctx context.Context, url, token string, out any) error {
	err := c.doJSON(ctx, http.MethodGet, url, token, nil, out)
	if hasStatus(err, http.StatusForbidden, http.StatusNotFound) {
		return nil
	}
	return err
}

// TRACEABILITY_MAPPING: REQ-015, REQ-047
func (c *Client) FetchTrafficSnapshot(ctx context.Context, token, owner, repo string) (TrafficSnapshot, error) {
	var viewsResp struct {
		Count   int `json:"count"`
		Uniques int `json:"uniques"`
	}
	viewsURL := fmt.Sprintf("%s/repos/%s/%s/traffic/views", c.APIBaseURL, owner, repo)
	if err := c.fetchTraffic(ctx, viewsURL, token, &viewsResp); err != nil {
		return TrafficSnapshot{}, err
	}

	var clonesResp struct {
		Count   int `json:"count"`
		Uniques int `json:"uniques"`
	}
	clonesURL := fmt.Sprintf("%s/repos/%s/%s/traffic/clones", c.APIBaseURL, owner, repo)
	if err := c.fetchTraffic(ctx, clonesURL, token, &clonesResp); err != nil {
		return TrafficSnapshot{}, err
	}

	var referrers []ReferrerItem
	refURL := fmt.Sprintf("%s/repos/%s/%s/traffic/popular/referrers", c.APIBaseURL, owner, repo)
	if err := c.fetchTraffic(ctx, refURL, token, &referrers); err != nil {
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
