package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	q.Set("secret_returned", "true")
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
