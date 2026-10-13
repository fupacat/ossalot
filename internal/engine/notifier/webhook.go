package notifier

import (
	"context"
	"net/http"

	"github.com/eric/ossalot/internal/db"
)

// TRACEABILITY_MAPPING: REQ-028
type Notifier struct {
	HTTPClient *http.Client
}

// TRACEABILITY_MAPPING: REQ-028
func NewNotifier(httpClient *http.Client) *Notifier {
	panic("stub: REQ-028 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-028
func (n *Notifier) NotifyHighValueOpportunity(ctx context.Context, settings db.WorkspaceSettings, opp db.Opportunity) (int, error) {
	panic("stub: REQ-028 not yet implemented")
}
