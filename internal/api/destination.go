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

type GetDestinationResponse struct {
	Total    int           `json:"total"`
	PageSize int           `json:"page_size"`
	Page     int           `json:"page"`
	Result   []Destination `json:"result"`
}

type Destination struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Connector       string         `json:"connector"`
	ConnectorStatus string         `json:"connector_status,omitempty"`
	Config          map[string]any `json:"config"`
	KcClusterId     string         `json:"kc_cluster_id,omitempty"`
}

func (s *streamkapAPI) CreateDestination(ctx context.Context, reqPayload Destination) (*Destination, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/destinations?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateDestination request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Destination
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			tflog.Info(ctx, fmt.Sprintf(
				"Destination %q already exists — attempting to adopt existing resource", reqPayload.Name))
			adopted, adoptErr := s.adoptDestinationByName(ctx, reqPayload.Name)
			if adoptErr == nil {
				return adopted, nil
			}
			// See matching comment in source.go CreateSource.
			return nil, fmt.Errorf("%w (also tried to adopt the existing resource but could not locate it via the list endpoint: %v)", err, adoptErr)
		}
		return nil, err
	}

	return &resp, nil
}

// adoptDestinationByName — see adoptSourceByName for rationale (paginates
// through all partial_name matches to find the exact-name destination).
func (s *streamkapAPI) adoptDestinationByName(ctx context.Context, name string) (*Destination, error) {
	const pageSize = 100
	const maxPages = 1000
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/destinations?secret_returned=true&page=%d&page_size=%d&partial_name=%s",
			s.cfg.BaseURL, page, pageSize, url.QueryEscape(name))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to build adopt request for %q: %w", name, err)
		}
		var resp GetDestinationResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, fmt.Errorf("failed to list destinations while adopting %q: %w", name, err)
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
	return nil, fmt.Errorf("destination %q reported as existing but not found in list", name)
}

func (s *streamkapAPI) GetDestination(ctx context.Context, destinationID string) (*Destination, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/destinations/"+destinationID+"?secret_returned=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetDestination request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetDestinationResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) ListDestinations(ctx context.Context) ([]Destination, error) {
	// Backend default page_size is 10 (max 100); iterate to return all pages.
	const pageSize = 100
	const maxPages = 1000
	var all []Destination
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/destinations?secret_returned=true&page=%d&page_size=%d", s.cfg.BaseURL, page, pageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf("ListDestinations request details:\n\tMethod: %s\n\tURL: %s\n", req.Method, req.URL.String()))
		var resp GetDestinationResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Result...)
		// Short-page termination only; Total can lie under concurrent
		// deletes (coderabbit PR #70 comment). maxPages caps runaway.
		if len(resp.Result) < pageSize {
			break
		}
	}
	return all, nil
}

func (s *streamkapAPI) DeleteDestination(ctx context.Context, destinationID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/destinations/"+destinationID+"?secret_returned=true&wait=false", http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteDestination request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Destination
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateDestination(ctx context.Context, destinationID string, reqPayload Destination) (*Destination, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/destinations/"+destinationID+"?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateDestination request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Destination
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
