package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SmtChainItem is one transform in a source's or destination's SMT chain.
// List order is execution order.
type SmtChainItem struct {
	// ID is the Kafka Connect alias (e.g. MaskField1); the server assigns one when it is empty.
	ID        string         `json:"id,omitempty"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	Enabled   bool           `json:"enabled"`
	Config    map[string]any `json:"config,omitempty"`
	Predicate *SmtPredicate  `json:"predicate,omitempty"`
}

// SmtPredicate scopes a chain item to the records it matches.
type SmtPredicate struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
	Negate bool           `json:"negate"`
}

// SmtCatalogEntry is a transform or predicate type a chain can use.
type SmtCatalogEntry struct {
	Type       string `json:"type"`
	Properties []struct {
		Name   string `json:"name"`
		Secret bool   `json:"secret"`
	} `json:"properties"`
}

// ListSmtCatalog lists the SMT types a chain on host ("sources" or
// "destinations") can use. It is empty while chains are off for the service.
func (s *streamkapAPI) ListSmtCatalog(ctx context.Context, host string) ([]SmtCatalogEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/"+host+"/smt-catalog", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("ListSmtCatalog request details:\n\tMethod: %s\n\tURL: %s\n", req.Method, req.URL.String()))
	var resp []SmtCatalogEntry
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
