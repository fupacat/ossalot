package api

import (
	"github.com/eric/ossalot/internal/engine/community"
	"github.com/eric/ossalot/internal/engine/notifier"
	"github.com/eric/ossalot/internal/scheduler"
)

// TRACEABILITY_MAPPING: REQ-030
func (s *Server) RegisterCommunityAndInboxRoutes(scanner *community.Scanner, notif *notifier.Notifier, runner *scheduler.Runner) {
	panic("stub: REQ-030 not yet implemented")
}
