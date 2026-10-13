package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-028
type Notifier struct {
	HTTPClient *http.Client
}

// TRACEABILITY_MAPPING: REQ-028
func NewNotifier(httpClient *http.Client) *Notifier {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Notifier{HTTPClient: httpClient}
}

// TRACEABILITY_MAPPING: REQ-028
func (n *Notifier) NotifyHighValueOpportunity(ctx context.Context, settings db.WorkspaceSettings, opp db.Opportunity) (int, error) {
	if opp.RelevanceScore < 85 {
		return 0, nil
	}
	targets := []string{}
	if settings.SlackWebhookURL != "" {
		targets = append(targets, settings.SlackWebhookURL)
	}
	if settings.DiscordWebhookURL != "" {
		targets = append(targets, settings.DiscordWebhookURL)
	}
	if len(targets) == 0 {
		return 0, nil
	}

	payload, _ := json.Marshal(map[string]any{
		"text":            fmt.Sprintf("[OSSAlot %d%%] %s (%s)", opp.RelevanceScore, opp.Title, opp.URL),
		"relevance_score": opp.RelevanceScore,
		"category":        opp.Category,
		"url":             opp.URL,
	})

	sent := 0
	for _, targetURL := range targets {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := n.HTTPClient.Do(req)
		if err != nil {
			return sent, err
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return sent, fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		sent++
	}
	return sent, nil
}
