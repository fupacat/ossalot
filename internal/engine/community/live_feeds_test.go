package community

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type offlineTransport struct{}

func (offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

func fetchFeed(t *testing.T, adapter *LiveFeedTransport, path string) []communityFeedItem {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "http://feeds.test"+path, nil)
	resp, err := adapter.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: expected a 200 response, got %v err=%v", path, resp, err)
	}
	var items []communityFeedItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("%s: invalid JSON: %v", path, err)
	}
	return items
}

func adapterFor(base http.RoundTripper, apiURL string) *LiveFeedTransport {
	return &LiveFeedTransport{Base: base, AlgoliaURL: apiURL, StackExchangeURL: apiURL, RedditURL: apiURL, DevToURL: apiURL}
}

// TRACEABILITY_MAPPING: REQ-050
func TestLiveFeedTransport(t *testing.T) {
	var queries []string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("tags") == "ask_hn" {
			_, _ = w.Write([]byte(`{"hits":[{"objectID":"9","title":"Ask HN: parsers?","author":"asker","points":12}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"hits":[{"objectID":"1","title":"Show HN: a parser","url":"https://example.com/p","author":"pg","points":120},{"objectID":"2","title":"","points":5}]}`))
	})
	mux.HandleFunc("GET /2.3/search/advanced", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"question_id":77,"title":"Parser error","link":"https://stackoverflow.com/q/77","owner":{"display_name":"dev"}}]}`))
	})
	mux.HandleFunc("GET /search.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"children":[{"data":{"id":"a","title":"Parsing help","permalink":"/r/golang/a","author":"gopher","subreddit_name_prefixed":"r/golang"}},{"data":{"id":"b","title":"No subreddit","permalink":"/b","author":"x"}},{"data":{"id":"c","title":""}}]}}`))
	})
	mux.HandleFunc("GET /api/articles", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		_, _ = w.Write([]byte(`[{"id":5,"title":"Writing a parser","url":"https://dev.to/x/5","user":{"username":"writer"}}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	adapter := adapterFor(http.DefaultTransport, srv.URL)

	hn := fetchFeed(t, adapter, "/feeds/hackernews?q=parser")
	if len(hn) != 1 || hn[0].Score != 91 || hn[0].Community != "Hacker News" || hn[0].URL != "https://example.com/p" {
		t.Fatalf("unexpected Hacker News items: %+v", hn)
	}
	ask := fetchFeed(t, adapter, "/feeds/lobsters?q=parser")
	if len(ask) != 1 || ask[0].Community != "Ask HN" || ask[0].Score != 82 || ask[0].URL != "https://news.ycombinator.com/item?id=9" {
		t.Fatalf("lobsters must be answered from Ask HN and labeled Ask HN: %+v", ask)
	}
	so := fetchFeed(t, adapter, "/feeds/stackoverflow?q=parser")
	if len(so) != 1 || so[0].Score != 88 || so[0].Community != "StackOverflow" || so[0].ID != "77" {
		t.Fatalf("unexpected Stack Overflow items: %+v", so)
	}
	reddit := fetchFeed(t, adapter, "/feeds/reddit?q=parser")
	if len(reddit) != 2 || reddit[0].Community != "r/golang" || reddit[1].Community != "Reddit" || reddit[0].Score != 86 || reddit[0].URL != "https://www.reddit.com/r/golang/a" {
		t.Fatalf("unexpected Reddit items: %+v", reddit)
	}
	devto := fetchFeed(t, adapter, "/feeds/devto?q=Code-Gen+tools")
	if len(devto) != 1 || devto[0].Score != 84 || devto[0].Community != "Dev.to" {
		t.Fatalf("unexpected Dev.to items: %+v", devto)
	}
	if !strings.Contains(queries[len(queries)-1], "tag=codegen") {
		t.Fatalf("the Dev.to tag must be the keyword's first word, lowercased without hyphens: %v", queries)
	}

	before := len(queries)
	fetchFeed(t, adapter, "/feeds/hackernews")
	if !strings.Contains(queries[before], "query=open+source") {
		t.Fatalf("an empty keyword must search for open source: %v", queries[before:])
	}
	if items := fetchFeed(t, adapter, "/feeds/myspace?q=parser"); len(items) != 0 {
		t.Fatalf("an unknown platform must return no items, got %+v", items)
	}
	t.Logf("platforms=5 lobsters_label=%s empty_q=open_source", strings.ReplaceAll(ask[0].Community, " ", "_"))
}

// TRACEABILITY_MAPPING: REQ-050
func TestLiveFeedTransportUnavailableSources(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	garbled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer garbled.Close()

	adapters := map[string]*LiveFeedTransport{
		"offline":  adapterFor(offlineTransport{}, "http://unreachable.test"),
		"status":   adapterFor(http.DefaultTransport, failing.URL),
		"bad json": adapterFor(http.DefaultTransport, garbled.URL),
	}
	for name, adapter := range adapters {
		for _, platform := range []string{"hackernews", "lobsters", "stackoverflow", "reddit", "devto"} {
			if items := fetchFeed(t, adapter, "/feeds/"+platform+"?q=parser"); len(items) != 0 {
				t.Fatalf("%s %s: an unavailable source must return no items, got %+v", name, platform, items)
			}
		}
	}
}
