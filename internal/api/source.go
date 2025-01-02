package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type GetSourceResponse struct {
	Total    int      `json:"total"`
	PageSize int      `json:"page_size"`
	Page     int      `json:"page"`
	Result   []Source `json:"result"`
}

type Source struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	Connector string         `json:"connector"`
	Config    map[string]any `json:"config"`
}

func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload Source) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/sources?secret_returned=true", bytes.NewBuffer(payload))
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
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
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

func (s *streamkapAPI) DeleteSource(ctx context.Context, sourceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/sources/"+sourceID+"?secret_returned=true", http.NoBody)
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
	err = s.doRequest(ctx, req, &resp)
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
		ctx, http.MethodPut, s.cfg.BaseURL+"/sources/"+sourceID+"?secret_returned=true", bytes.NewBuffer(payload))
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
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
