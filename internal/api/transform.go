package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

type GetTransformResponse struct {
	Total    int         `json:"total"`
	PageSize int         `json:"page_size"`
	Page     int         `json:"page"`
	Result   []Transform `json:"result"`
}

// Transform represents the API response structure for transforms
type Transform struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name"`
	TransformType  string         `json:"transform_type"`
	Status         string         `json:"status"`
	Config         map[string]any `json:"config"`
	Implementation map[string]any `json:"implementation"`
	Version        int            `json:"version"`
	TopicIDs       []string       `json:"topic_ids"`
	Topics         []string       `json:"topics"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	StartTime      *string        `json:"start_time"`
}

// CreateTransformRequest represents the request payload for creating a transform
type CreateTransformRequest struct {
	Transform   string         `json:"transform"`
	Config      map[string]any `json:"config"`
	CreatedFrom string         `json:"created_from,omitempty"`
}

// UpdateTransformRequest represents the request payload for updating a transform
type UpdateTransformRequest struct {
	ID             string         `json:"id"`
	Transform      string         `json:"transform"`
	Config         map[string]any `json:"config"`
	Implementation map[string]any `json:"implementation,omitempty"`
}

func (s *streamkapAPI) GetTransform(ctx context.Context, TransformID string) (*Transform, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/transforms/"+TransformID+"?secret_returned=true&unwind_topics=false", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetTransformResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) ListTransforms(ctx context.Context) ([]Transform, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/transforms?secret_returned=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"ListTransforms request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetTransformResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (s *streamkapAPI) CreateTransform(ctx context.Context, reqPayload CreateTransformRequest) (*Transform, error) {
	// Set created_from to terraform
	reqPayload.CreatedFrom = constants.TERRAFORM

	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/transforms?secret_returned=true", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreateTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Transform
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) UpdateTransform(ctx context.Context, transformID string, reqPayload UpdateTransformRequest) (*Transform, error) {
	// Ensure ID is set in the request body (transforms use ID in body, not URL path)
	reqPayload.ID = transformID

	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	// Note: Transforms use PUT /transforms (no ID in path), ID is in request body
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/transforms?secret_returned=true", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Transform
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) DeleteTransform(ctx context.Context, transformID string) error {
	// Note: Transforms use DELETE /transforms?id={id} (query param, not path param)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/transforms?id="+transformID, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeleteTransform request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Transform
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}
