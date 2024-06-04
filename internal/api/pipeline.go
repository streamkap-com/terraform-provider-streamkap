package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Pipeline struct {
	ID                string              `json:"id,omitempty"`
	Name              string              `json:"name"`
	SnapshotNewTables bool                `json:"snapshot_new_tables"`
	Source            PipelineSource      `json:"source"`
	Destination       PipelineDestination `json:"destination"`
	Transforms        []string            `json:"transforms"`
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

func (s *streamkapAPI) CreatePipeline(ctx context.Context, reqPayload Pipeline) (*Pipeline, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/pipelines?secret_returned=true", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Pipeline
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/pipelines?secret_returned=true&id="+pipelineID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp GetPipelineResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) DeletePipeline(ctx context.Context, pipelineID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/pipelines?secret_returned=true&id="+pipelineID, http.NoBody)
	if err != nil {
		return err
	}
	var resp Pipeline
	err = s.doRequest(req, &resp)
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
		ctx, http.MethodPut, s.cfg.BaseURL+"/api/pipelines?secret_returned=true&id="+pipelineID, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Pipeline
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
