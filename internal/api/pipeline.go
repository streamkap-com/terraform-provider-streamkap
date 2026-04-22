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

type Pipeline struct {
	ID                string               `json:"id,omitempty"`
	Name              string               `json:"name"`
	SnapshotNewTables bool                 `json:"snapshot_new_tables"`
	Source            PipelineSource       `json:"source"`
	Destination       PipelineDestination  `json:"destination"`
	Transforms        []*PipelineTransform `json:"transforms"`
	Tags              []string             `json:"tags"`
}

type GetPipelineResponse struct {
	Total    int        `json:"total"`
	PageSize int        `json:"page_size"`
	Page     int        `json:"page"`
	Result   []Pipeline `json:"result"`
}

type PipelineSource struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Connector string   `json:"connector"`
	Topics    []string `json:"topics"`
}

type PipelineDestination struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connector string `json:"connector"`
}

type PipelineTransform struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	StartTime *string `json:"start_time"`
	TopicID   string  `json:"topic_id"`
	Topic     string  `json:"topic"`
}

func (s *streamkapAPI) CreatePipeline(ctx context.Context, reqPayload Pipeline) (*Pipeline, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/pipelines?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"CreatePipeline request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		// Create-or-adopt: if a pipeline with this name already exists (409 from a
		// previous timed-out create), adopt the existing one instead of failing.
		if strings.Contains(err.Error(), "already exists") {
			tflog.Info(ctx, fmt.Sprintf(
				"Pipeline %q already exists — adopting existing resource", reqPayload.Name))
			return s.adoptPipelineByName(ctx, reqPayload.Name)
		}
		return nil, err
	}

	return &resp, nil
}

// adoptPipelineByName finds an existing pipeline by name and returns it,
// allowing Terraform to adopt it into state after a 409/422 conflict.
// The pipelines endpoint only supports a `partial_name` filter, so we pass
// the name through that filter (max page_size=100) and then match exactly
// client-side. Without this, the default page_size of 10 would silently
// miss pipelines on later pages.
func (s *streamkapAPI) adoptPipelineByName(ctx context.Context, name string) (*Pipeline, error) {
	reqURL := s.cfg.BaseURL + "/pipelines?secret_returned=true&page_size=100&partial_name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to build adopt request for %q: %w", name, err)
	}
	var resp GetPipelineResponse
	if err := s.doRequest(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list pipelines while adopting %q: %w", name, err)
	}
	for i := range resp.Result {
		if resp.Result[i].Name == name {
			return &resp.Result[i], nil
		}
	}
	return nil, fmt.Errorf("pipeline %q reported as existing but not found in list", name)
}

func (s *streamkapAPI) GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/pipelines/"+pipelineID+"?secret_returned=true", http.NoBody)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"GetPipeline request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp GetPipelineResponse
	err = s.doRequest(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, nil
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) ListPipelines(ctx context.Context) ([]Pipeline, error) {
	// Backend default page_size is 10 (max 100); iterate to return all pages.
	const pageSize = 100
	var all []Pipeline
	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s/pipelines?secret_returned=true&page=%d&page_size=%d", s.cfg.BaseURL, page, pageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return nil, err
		}
		tflog.Debug(ctx, fmt.Sprintf("ListPipelines request details:\n\tMethod: %s\n\tURL: %s\n", req.Method, req.URL.String()))
		var resp GetPipelineResponse
		if err := s.doRequest(ctx, req, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Result...)
		if len(resp.Result) < pageSize || len(all) >= resp.Total {
			break
		}
	}
	return all, nil
}

func (s *streamkapAPI) DeletePipeline(ctx context.Context, pipelineID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/pipelines/"+pipelineID+"?secret_returned=true&wait=false", http.NoBody)
	if err != nil {
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"DeletePipeline request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n",
		req.Method,
		req.URL.String(),
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdatePipeline(ctx context.Context, pipelineID string, reqPayload Pipeline) (*Pipeline, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/pipelines/"+pipelineID+"?secret_returned=true&wait=false", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf(
		"UpdatePipeline request details:\n"+
			"\tMethod: %s\n"+
			"\tURL: %s\n"+
			"\tBody: %s",
		req.Method,
		req.URL.String(),
		payload,
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
