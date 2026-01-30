package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

type GetTagResponse struct {
	Tags []Tag `json:"tags"`
}

type Tag struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        []string `json:"type"`
	System      bool     `json:"system"`
	Custom      *bool    `json:"custom"`
}

func (s *streamkapAPI) GetTag(ctx context.Context, TagID string) (*Tag, error) {
	url, err := url.Parse(s.cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	url = url.JoinPath("tags")
	q := url.Query()
	q.Set("tag_ids", TagID)
	url.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetTagResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Tags) == 0 {
		return nil, nil
	}

	return &resp.Tags[0], nil
}

func (s *streamkapAPI) CreateTag(ctx context.Context, reqPayload Tag) (*Tag, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/tags", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) UpdateTag(ctx context.Context, tagID string, reqPayload Tag) (*Tag, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/tags/"+tagID, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *streamkapAPI) DeleteTag(ctx context.Context, tagID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/tags/"+tagID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteTag request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Tag
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}
	return nil
}
