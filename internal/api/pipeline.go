package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type CreatePipelineRequest struct {
	ID          string                    `json:"-"`
	Name        string                    `json:"name"`
	Destination CreatePipelineDestination `json:"destination"`
	Source      CreatePipelineSource      `json:"source"`
	Transforms  []string                  `json:"transforms"`
}

type PipelineResponse struct {
	Total    int        `json:"total"`
	PageSize int        `json:"page_size"`
	Page     int        `json:"page"`
	Result   []Pipeline `json:"result"`
}

type CreatePipelineDestination struct {
	Connector string `json:"connector"`
	Name      string `json:"name"`
	ID        struct {
		OID string `json:"$oid"`
	} `json:"id"`
}

type CreatePipelineSource struct {
	Connector string   `json:"connector"`
	Name      string   `json:"name"`
	Topics    []string `json:"topics"`
	ID        struct {
		OID string `json:"$oid"`
	} `json:"id"`
}

type Pipeline struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	SubID    string   `json:"sub_id"`
	TenantID string   `json:"tenant_id"`
	TopicIds []string `json:"topic_ids"`
	Source   struct {
		Name            string `json:"name"`
		Connector       string `json:"connector"`
		ID              string `json:"id"`
		Tasks           []int  `json:"tasks"`
		ConnectorStatus string `json:"connector_status"`
		TaskStatuses    struct {
			Field1 struct {
				Status string `json:"status"`
			} `json:"0"`
		} `json:"task_statuses"`
		SnapshotState  string `json:"snapshot_state"`
		SnapshotStatus []struct {
			Status          string `json:"status"`
			Topic           string `json:"topic"`
			SubmitTimestamp string `json:"submit_timestamp"`
		} `json:"snapshot_status"`
		Topics []string `json:"topics"`
	} `json:"source"`
	Destination struct {
		Name            string `json:"name"`
		Connector       string `json:"connector"`
		ID              string `json:"id"`
		Tasks           []int  `json:"tasks"`
		ConnectorStatus string `json:"connector_status"`
		TaskStatuses    struct {
			Field1 struct {
				Status string `json:"status"`
			} `json:"0"`
		} `json:"task_statuses"`
	} `json:"destination"`
	Transforms []interface{} `json:"transforms"`
}

func (s *streamkapAPI) CreatePipeline(ctx context.Context, reqPayload CreatePipelineRequest) (*Pipeline, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/pipelines", bytes.NewBuffer(payload))
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

func (s *streamkapAPI) GetPipeline(ctx context.Context, pipelineID string) ([]Pipeline, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/pipelines?id="+pipelineID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp PipelineResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (s *streamkapAPI) DeletePipeline(ctx context.Context, pipelineID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/pipelines?id="+pipelineID, http.NoBody)
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

func (s *streamkapAPI) UpdatePipeline(ctx context.Context, reqPayload CreatePipelineRequest) (*Pipeline, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/api/pipelines?id="+reqPayload.ID, bytes.NewBuffer(payload))
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
