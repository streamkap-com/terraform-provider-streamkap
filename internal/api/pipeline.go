package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
		// Do NOT auto-adopt pipelines on 409. Streamkap enforces unique
		// {tenant_id, service_id, name} for pipelines, so a 409 means one
		// already exists with this name. Two scenarios produce it:
		//
		//   1. A previous apply created the pipeline but lost the response
		//      (network, timeout) — Terraform's state thinks the resource
		//      doesn't exist but the backend has it. Recovery: `terraform
		//      import`.
		//
		//   2. A `lifecycle { create_before_destroy = true }` replace
		//      operation is in flight: Terraform created a *deposed* state
		//      entry from the old live instance (id=X), then asked the
		//      provider to create the *new* live instance, which collides
		//      on name with the still-existing backend pipeline X.
		//
		// An earlier version of this code adopted the existing pipeline by
		// name and returned it as the create result. That is *unsafe* under
		// scenario 2: the new live state entry would end up with the same
		// backend id as the deposed entry, and the next step of the apply
		// (destroy the deposed) would DELETE the very pipeline we just
		// "adopted", leaving the live state pointing at a tombstone. We
		// confirmed this trace against a real customer log
		// (0_Terraform Apply.txt) — they had 4 deposed entries all pointing
		// to the same backend pipeline because of repeated adoption.
		//
		// We can't disambiguate the two scenarios from inside the API
		// client (no state visibility) and a content-comparison adopt is
		// also unsafe (taint + create_before_destroy + no config change
		// collapses to the same data-loss). So we surface a clear error
		// and leave recovery to the user. Loud failure beats silent data
		// loss.
		if strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf(
				"streamkap_pipeline %q already exists on the backend, and auto-adoption is unsafe for this resource (would risk destroying the live pipeline under create_before_destroy). Recovery options:\n"+
					"  • If this is a `lifecycle { create_before_destroy = true }` replace: remove that directive — Streamkap enforces unique pipeline names per tenant/service so a new and an old pipeline cannot coexist by name. Use the default destroy-then-create, or change the pipeline name so old and new can briefly coexist.\n"+
					"  • If you already have deposed entries from previous failed applies (visible in `terraform plan` as `<address> (destroy deposed <key>)`), they point at the same backend record and must be removed from state before any retry will work: `terraform state rm '<resource_address>'` (Terraform will refuse the deposed-key bit automatically; the address alone removes both live and deposed copies). See docs/MIGRATION.md → \"Known limitations\".\n"+
					"  • If this is recovering from a previous apply that lost its response: run `terraform import streamkap_pipeline.<resource_name> <pipeline_id>` (find the id via the Streamkap UI or `GET /pipelines?partial_name=%s`) and re-run apply.\n"+
					"Original backend error: %w",
				reqPayload.Name, reqPayload.Name, err,
			)
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
		redactSensitiveJSON(payload),
	))
	var resp Pipeline
	err = s.doRequestWithRetry(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
