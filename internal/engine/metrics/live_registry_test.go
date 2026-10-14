package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

// TRACEABILITY_MAPPING: REQ-049
func TestLiveRegistryTransport(t *testing.T) {
	npm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/downloads/point/last-day/left-pad" {
			_, _ = w.Write([]byte(`{"downloads":1234,"package":"left-pad"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer npm.Close()

	client := NewRegistryClient("http://registry.test", &http.Client{
		Transport: &LiveRegistryTransport{Base: http.DefaultTransport, NPMBaseURL: npm.URL},
	})
	got := map[string]int{}
	for name, pkgs := range map[string]map[string]string{
		"left-pad": {"npm": "left-pad"},
		"missing":  {"npm": "missing-pkg"},
		"scoped":   {"npm": "@scope/pkg"},
		"pypi":     {"pypi": "requests"},
	} {
		downloads, err := client.FetchDownloads(context.Background(), pkgs)
		if err != nil {
			t.Fatalf("%s: every adapter response must be 200, got %v", name, err)
		}
		for _, n := range downloads {
			got[name] = n
		}
	}
	if got["left-pad"] != 1234 || got["missing"] != 0 || got["scoped"] != 0 || got["pypi"] != 0 {
		t.Fatalf("unexpected download counts: %v", got)
	}
	t.Logf("left-pad=%d missing=%d scoped=%d pypi=%d", got["left-pad"], got["missing"], got["scoped"], got["pypi"])

	// Paths outside the /downloads/{ecosystem}/{package} contract report 0
	adapter := &LiveRegistryTransport{Base: http.DefaultTransport, NPMBaseURL: npm.URL}
	for _, path := range []string{"/downloads/npm", "/stats/npm/left-pad"} {
		req, _ := http.NewRequest(http.MethodGet, "http://registry.test"+path, nil)
		resp, err := adapter.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected a 200 response, got %v err=%v", path, resp, err)
		}
		var body struct {
			Downloads int `json:"downloads"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.Downloads != 0 {
			t.Fatalf("%s: expected 0 downloads, got %d", path, body.Downloads)
		}
	}

	// A failed lookup reports 0
	offline := NewRegistryClient("http://registry.test", &http.Client{
		Transport: &LiveRegistryTransport{Base: failingTransport{}, NPMBaseURL: npm.URL},
	})
	downloads, err := offline.FetchDownloads(context.Background(), map[string]string{"npm": "left-pad"})
	if err != nil || downloads["npm"] != 0 {
		t.Fatalf("expected 0 downloads without error when npm is unreachable, got %v err=%v", downloads, err)
	}
}
