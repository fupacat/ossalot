package api

import (
	"net/http"

	"github.com/eric/ossalot/internal/db"
	"github.com/eric/ossalot/internal/github"
	"github.com/eric/ossalot/internal/llm"
)

// TRACEABILITY_MAPPING: REQ-011
type Server struct {
	store    *db.Store
	ghClient *github.Client
	llm      *llm.Client
	mux      *http.ServeMux
}

// TRACEABILITY_MAPPING: REQ-011
func NewServer(store *db.Store, ghClient *github.Client, llmClient *llm.Client) *Server {
	panic("stub: REQ-011 not yet implemented")
}

// TRACEABILITY_MAPPING: REQ-011
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("stub: REQ-011 not yet implemented")
}
