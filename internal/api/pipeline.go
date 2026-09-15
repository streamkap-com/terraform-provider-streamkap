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

type Pipeline struct {
	ID                           string                                `json:"id,omitempty"`
	Name                         string                                `json:"name"`
	SnapshotNewTables            bool                                  `json:"snapshot_new_tables"`
	Source                       PipelineSource                        `json:"source"`
	Destination                  PipelineDestination                   `json:"destination"`
	Transforms                   []*PipelineTransform                  `json:"transforms"`
	TopicAutoDiscoveryTransforms []PipelineTopicAutoDiscoveryTransform `json:"topic_auto_discovery_transforms"`
	Tags                         []string                              `json:"tags"`

	// PeriodicAudit is configured outside Terraform (the UI), but it has to be
	// round-tripped on update: the backend lists periodic_audit in
	// _CONDITIONAL_ENTITY_FIELDS, so a PUT that omits the key $unsets it and the
	// audit config is lost. Read returns topics as pretty names and update
	// expects the same, so the value echoes back unchanged.
	PeriodicAudit *PipelinePeriodicAudit `json:"periodic_audit,omitempty"`
}

// PipelinePeriodicAudit mirrors the backend's periodic row-level audit config.
type PipelinePeriodicAudit struct {
	Topics          []string `json:"topics"`
	TimestampColumn string   `json:"timestamp_column"`
	IntervalMinutes int64    `json:"interval_minutes"`
	FixDeletesOnly  *bool    `json:"fix_deletes_only,omitempty"`
}

// PipelineTopicAutoDiscoveryTransform lets a pipeline auto-discover a
// transform's OUTPUT topics by regex. Transform output topic names are
// generated dynamically (e.g. by a topic-router transform) and are not known
// at infra time, so they cannot be enumerated in transforms[].topics up front.
// The backend resolves the regex against live topics server-side and echoes
// this list back unchanged on GET.
type PipelineTopicAutoDiscoveryTransform struct {
	TransformID string `json:"transform_id"`
	Regex       string `json:"regex"`
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
		redactSensitiveJSON(payload),
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		// Pipelines answer this collision with a 409 rather than a 422; the
		// refusal is the same. See adoptRefusedError.
		if isAlreadyExists(err) {
			return nil, adoptRefusedError(adoptConflict{
				Kind:        "pipeline",
				Name:        reqPayload.Name,
				UniqueScope: "tenant/service",
				ImportAddr:  "streamkap_pipeline.<resource_name> <pipeline_id>",
				ListPath:    "/pipelines",
			}, err)
		}
		return nil, err
	}

	return &resp, nil
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
	const maxPages = 1000
	var all []Pipeline
	for page := 1; page <= maxPages; page++ {
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
		// Short-page termination only; Total can lie under concurrent
		// deletes (coderabbit PR #70 comment). maxPages caps runaway.
		if len(resp.Result) < pageSize {
			break
		}
	}
	return all, nil
}

func (s *streamkapAPI) DeletePipeline(ctx context.Context, pipelineID string) error {
	return s.deleteResource(ctx, "DeletePipeline", s.cfg.BaseURL+"/pipelines/"+pipelineID+"?secret_returned=true&wait=false")
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
		redactSensitiveJSON(payload),
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
