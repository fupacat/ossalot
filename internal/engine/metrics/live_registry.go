package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// LiveRegistryTransport answers the RegistryClient's /downloads/{ecosystem}/{package} contract
// from public registries. Only unscoped npm packages are looked up; everything else reports 0.
// TRACEABILITY_MAPPING: REQ-049
type LiveRegistryTransport struct {
	Base       http.RoundTripper
	NPMBaseURL string
}

// TRACEABILITY_MAPPING: REQ-049
func (t *LiveRegistryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	downloads := 0
	parts := strings.SplitN(strings.Trim(req.URL.Path, "/"), "/", 3)
	if len(parts) == 3 && parts[0] == "downloads" && parts[1] == "npm" && !strings.Contains(parts[2], "/") {
		downloads = t.npmDownloads(req.Context(), parts[2])
	}
	raw, _ := json.Marshal(map[string]int{"downloads": downloads})
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
		Request:    req,
	}, nil
}

// TRACEABILITY_MAPPING: REQ-049
func (t *LiveRegistryTransport) npmDownloads(ctx context.Context, pkg string) int {
	apiURL := fmt.Sprintf("%s/downloads/point/last-day/%s", t.NPMBaseURL, url.PathEscape(pkg))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	resp, err := t.Base.RoundTrip(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var data struct {
		Downloads int `json:"downloads"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&data)
	return data.Downloads
}
