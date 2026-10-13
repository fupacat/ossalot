package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TRACEABILITY_MAPPING: REQ-008
func TestGenerateFileTemplatesAndLLM(t *testing.T) {
	client := NewClient("", nil)
	ctx := context.Background()

	keys := []string{
		"contributing_guide",
		"code_of_conduct",
		"security_policy",
		"license_valid",
		"issue_templates",
		"pr_template",
		"ci_workflows",
		"readme_comprehensive",
	}
	for _, k := range keys {
		text, err := client.GenerateFile(ctx, GenerateRequest{
			RepoFullName:    "octocat/hello-world",
			RepoDescription: "Sample project",
			CheckKey:        k,
		})
		if err != nil || !strings.Contains(text, "octocat/hello-world") {
			t.Fatalf("template generation failed for %s: err=%v text=%q", k, err, text)
		}
	}

	// Test with explicit PrimaryLanguage
	withLang, err := client.GenerateFile(ctx, GenerateRequest{
		RepoFullName:    "octocat/hello-world",
		PrimaryLanguage: "Go",
		CheckKey:        "contributing_guide",
	})
	if err != nil || !strings.Contains(withLang, "(Go)") {
		t.Fatalf("expected Go in contributing guide: %q", withLang)
	}

	// Test remote LLM endpoint success and failure modes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /generate", func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Bearer sk-valid":
			_, _ = w.Write([]byte(`{"content":"# Custom AI Security Policy"}`))
		case "Bearer sk-empty":
			_, _ = w.Write([]byte(`{"content":""}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	remoteClient := NewClient(srv.URL+"/generate", srv.Client())
	aiOut, err := remoteClient.GenerateFile(ctx, GenerateRequest{
		Provider:     "openai",
		APIKey:       "sk-valid",
		RepoFullName: "octocat/hello-world",
		CheckKey:     "security_policy",
	})
	if err != nil || aiOut != "# Custom AI Security Policy" {
		t.Fatalf("remote LLM failed: %v, %q", err, aiOut)
	}

	if _, err := remoteClient.GenerateFile(ctx, GenerateRequest{APIKey: "sk-bad"}); err == nil {
		t.Fatalf("expected error on 401 LLM response")
	}
	if _, err := remoteClient.GenerateFile(ctx, GenerateRequest{APIKey: "sk-empty"}); err == nil {
		t.Fatalf("expected error on empty LLM response content")
	}

	badClient := NewClient("http://127.0.0.1:1/generate", srv.Client())
	if _, err := badClient.GenerateFile(ctx, GenerateRequest{APIKey: "sk-valid"}); err == nil {
		t.Fatalf("expected network error on closed port")
	}
}
