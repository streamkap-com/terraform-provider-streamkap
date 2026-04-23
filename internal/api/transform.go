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
	// Backend default page_size is 10 (max 100). Iterate until we have all
	// results so callers (sweepers, adopt fallbacks) see the full tenant, not
	// just page 1. See ListSources for the equivalent pattern.
	const pageSize = 100
	const maxPages = 1000
	var all []Transform
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/transforms?secret_returned=true&page=%d&page_size=%d", s.cfg.BaseURL, page, pageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf("ListTransforms request details:\n\tMethod: %s\n\tURL: %s\n", req.Method, req.URL.String()))
		var resp GetTransformResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Result...)
		if len(resp.Result) < pageSize {
			break
		}
	}
	return all, nil
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
		redactSensitiveJSON(payload),
	))
	var resp Transform
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			// The transform name lives inside the config map under the
			// backend's alias "transforms.name" (see base.go Create), not at
			// the top level like sources/destinations.
			name, _ := reqPayload.Config["transforms.name"].(string)
			tflog.Info(ctx, fmt.Sprintf(
				"Transform %q already exists — attempting to adopt existing resource", name))
			adopted, adoptErr := s.adoptTransformByName(ctx, name)
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

// adoptTransformByName — see adoptSourceByName for rationale (paginates
// through all partial_name matches to find the exact-name transform).
func (s *streamkapAPI) adoptTransformByName(ctx context.Context, name string) (*Transform, error) {
	const pageSize = 100
	const maxPages = 1000
	for page := 1; page <= maxPages; page++ {
		reqURL := fmt.Sprintf("%s/transforms?secret_returned=true&page=%d&page_size=%d&partial_name=%s",
			s.cfg.BaseURL, page, pageSize, url.QueryEscape(name))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("failed to build adopt request for %q: %w", name, err)
		}
		var resp GetTransformResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, fmt.Errorf("failed to list transforms while adopting %q: %w", name, err)
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
	return nil, fmt.Errorf("transform %q reported as existing but not found in list", name)
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
		redactSensitiveJSON(payload),
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

// TransformJobStatus represents the deployment status of a transform's Flink job
type TransformJobStatus struct {
	ID        string `json:"id"`
	JobName   string `json:"job_name"`
	Status    string `json:"status"`
	StartTime string `json:"start_time"`
}

// TransformImplementationDetails represents the implementation details for a transform
type TransformImplementationDetails struct {
	TransformID    string         `json:"transform_id"`
	VersionID      string         `json:"version_id,omitempty"`
	Description    *string        `json:"description,omitempty"`
	Implementation map[string]any `json:"implementation,omitempty"`
	Config         map[string]any `json:"config,omitempty"`
}

// TransformImplementationDetailsResponse represents the response from GET implementation_details
type TransformImplementationDetailsResponse struct {
	TransformID  string                                `json:"transform_id"`
	ImplVersions map[string]TransformImplementationDetails `json:"impl_versions"`
}

// GetTransformImplementationDetails retrieves implementation details including version history
func (s *streamkapAPI) GetTransformImplementationDetails(ctx context.Context, transformID string) (*TransformImplementationDetailsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/transforms/"+transformID+"/implementation_details", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTransformImplementationDetails request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp TransformImplementationDetailsResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateTransformImplementationDetails updates the implementation details for a transform
func (s *streamkapAPI) UpdateTransformImplementationDetails(ctx context.Context, transformID string, details TransformImplementationDetails) (*TransformImplementationDetails, error) {
	// Ensure transform_id is set
	details.TransformID = transformID

	payload, err := json.Marshal(details)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.cfg.BaseURL+"/transforms/"+transformID+"/implementation_details", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdateTransformImplementationDetails request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		redactSensitiveJSON(payload),
	))
	var resp TransformImplementationDetails
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeployTransformPreview triggers a preview deployment for a transform.
// versionID should be "no-version" (backend normalizes to "0").
// replayWindow examples: "7d", "24h", "10m", "0", "" (empty = latest offset).
func (s *streamkapAPI) DeployTransformPreview(ctx context.Context, transformID, versionID, replayWindow string) error {
	reqURL := s.cfg.BaseURL + "/transforms/" + transformID + "/deploy-job-preview/" + versionID
	if replayWindow != "" {
		reqURL += "?replay_window=" + url.QueryEscape(replayWindow)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeployTransformPreview request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp map[string]any
	return s.doRequestWithRetry(ctx, req, &resp)
}

// DeployTransformLive triggers a live deployment for a transform.
// versionID should be "no-version".
func (s *streamkapAPI) DeployTransformLive(ctx context.Context, transformID, versionID string) error {
	reqURL := s.cfg.BaseURL + "/transforms/" + transformID + "/deploy-job-live/" + versionID

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeployTransformLive request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp map[string]any
	return s.doRequestWithRetry(ctx, req, &resp)
}

// GetTransformJobStatus retrieves the current Flink job status for a transform.
func (s *streamkapAPI) GetTransformJobStatus(ctx context.Context, transformID string) (*TransformJobStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/transforms/"+transformID+"/job_status", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetTransformJobStatus request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp TransformJobStatus
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
