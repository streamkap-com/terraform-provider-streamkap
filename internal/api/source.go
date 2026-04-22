package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

type GetSourceResponse struct {
	Total    int      `json:"total"`
	PageSize int      `json:"page_size"`
	Page     int      `json:"page"`
	Result   []Source `json:"result"`
}

type Source struct {
	ID              string         `json:"id,omitempty"`
	Name            string         `json:"name"`
	Connector       string         `json:"connector"`
	ConnectorStatus string         `json:"connector_status,omitempty"`
	Config          map[string]any `json:"config"`
	KcClusterId     string         `json:"kc_cluster_id,omitempty"`
}

func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload Source) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]any
	err = json.Unmarshal(payload, &payloadMap)
    if err != nil {
        return nil, err
    }

    payloadMap["created_from"] = constants.TERRAFORM

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/sources?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateSource request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Source
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			tflog.Info(ctx, fmt.Sprintf(
				"Source %q already exists — adopting existing resource", reqPayload.Name))
			return s.adoptSourceByName(ctx, reqPayload.Name)
		}
		return nil, err
	}

	return &resp, nil
}

// adoptSourceByName finds an existing source by name and returns it,
// allowing Terraform to adopt it into state after a 409/422 conflict.
// The /sources endpoint only accepts a `partial_name` filter — there is no
// exact-name filter — so we narrow server-side with partial_name and match
// exactly client-side. Iterates all pages: a single prefix can legitimately
// match many sources, and stopping at page 1 would miss the exact match.
func (s *streamkapAPI) adoptSourceByName(ctx context.Context, name string) (*Source, error) {
	const pageSize = 100
	const maxPages = 1000 // hard cap as a runaway safeguard
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/sources?secret_returned=true&page=%d&page_size=%d&partial_name=%s",
			s.cfg.BaseURL, page, pageSize, url.QueryEscape(name))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to build adopt request for %q: %w", name, err)
		}
		var resp GetSourceResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, fmt.Errorf("failed to list sources while adopting %q: %w", name, err)
		}
		for i := range resp.Result {
			if resp.Result[i].Name == name {
				return &resp.Result[i], nil
			}
		}
		if len(resp.Result) < pageSize {
			break
		}
	}
	return nil, fmt.Errorf("source %q reported as existing but not found in list", name)
}

func (s *streamkapAPI) GetSource(ctx context.Context, sourceID string) (*Source, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/sources/"+sourceID+"?secret_returned=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetSource request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetSourceResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) ListSources(ctx context.Context) ([]Source, error) {
	// Backend default page_size is 10 (max 100). Iterate until we have all
	// results so callers (adopt, sweepers) see the full tenant, not just page 1.
	const pageSize = 100
	const maxPages = 1000 // safeguard against runaway if Total/page_size semantics drift
	var all []Source
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/sources?secret_returned=true&page=%d&page_size=%d", s.cfg.BaseURL, page, pageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf("ListSources request details:\n\tMethod: %s\n\tURL: %s\n", req.Method, req.URL.String()))
		var resp GetSourceResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Result...)
		// Terminate only on a short page. `resp.Total` can lie under
		// concurrent deletes or when the server returns 0 inconsistently,
		// and using it as a second termination clause risks truncating
		// the result (coderabbit PR #70 comment). maxPages above caps
		// runaway if the server consistently returns full pages.
		if len(resp.Result) < pageSize {
			break
		}
	}
	return all, nil
}

func (s *streamkapAPI) DeleteSource(ctx context.Context, sourceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/sources/"+sourceID+"?secret_returned=true&wait=false", http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteSource request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Source
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateSource(ctx context.Context, sourceID string, reqPayload Source) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/sources/"+sourceID+"?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateSource request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Source
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
