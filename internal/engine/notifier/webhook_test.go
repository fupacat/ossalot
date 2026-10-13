package notifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-028
func TestNotifyHighValueOpportunity(t *testing.T) {
	defNotif := NewNotifier(nil)
	if defNotif.HTTPClient == nil {
		t.Fatalf("expected default HTTPClient")
	}

	calls := 0
	failWebhook := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failWebhook {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notif := NewNotifier(srv.Client())
	ctx := context.Background()
	settings := db.WorkspaceSettings{
		SlackWebhookURL:   srv.URL + "/slack",
		DiscordWebhookURL: srv.URL + "/discord",
	}

	// 1. Score < 85 sends 0 requests
	sent, err := notif.NotifyHighValueOpportunity(ctx, settings, db.Opportunity{
		Title:          "Moderate match",
		RelevanceScore: 75,
	})
	if err != nil || sent != 0 || calls != 0 {
		t.Fatalf("expected 0 notifications for score 75, got sent=%d err=%v", sent, err)
	}

	// 2. Empty webhook settings sends 0 requests
	sentEmpty, err := notif.NotifyHighValueOpportunity(ctx, db.WorkspaceSettings{}, db.Opportunity{
		Title:          "High match",
		RelevanceScore: 95,
	})
	if err != nil || sentEmpty != 0 {
		t.Fatalf("expected 0 notifications when no webhooks configured, got %d", sentEmpty)
	}

	// 3. Score >= 85 with Slack and Discord configured sends 2 requests
	sentHigh, err := notif.NotifyHighValueOpportunity(ctx, settings, db.Opportunity{
		Title:          "High match",
		URL:            "https://github.com/acme/app/issues/1",
		Category:       "github_issue",
		RelevanceScore: 90,
	})
	if err != nil || sentHigh != 2 || calls != 2 {
		t.Fatalf("expected 2 notifications sent, got sent=%d calls=%d err=%v", sentHigh, calls, err)
	}

	// 4. Non-200 webhook error
	failWebhook = true
	if _, err := notif.NotifyHighValueOpportunity(ctx, settings, db.Opportunity{RelevanceScore: 90}); err == nil {
		t.Fatalf("expected error on 500 webhook response")
	}

	// 5. Network failure error
	if _, err := notif.NotifyHighValueOpportunity(ctx, db.WorkspaceSettings{SlackWebhookURL: "http://127.0.0.1:1/slack"}, db.Opportunity{RelevanceScore: 90}); err == nil {
		t.Fatalf("expected network error on closed port")
	}
}
