package github

import "context"

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
	panic("stub: REQ-015 not yet implemented")
}
