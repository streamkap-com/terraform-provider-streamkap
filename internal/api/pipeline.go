package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type CreatePipelineRequest struct {
	Name        string `json:"name"`
	Destination struct {
		Connector string `json:"connector"`
		Name      string `json:"name"`
		Id        struct {
			Oid string `json:"$oid"`
		} `json:"id"`
	} `json:"destination"`
	Source struct {
		Connector string   `json:"connector"`
		Name      string   `json:"name"`
		Topics    []string `json:"topics"`
		Id        struct {
			Oid string `json:"$oid"`
		} `json:"id"`
	} `json:"source"`
	Transforms []interface{} `json:"transforms"`
}

type CreatePipelineResponse struct{}

func (s *streamkapAPI) CreatePipeline(ctx context.Context, reqPayload CreatePipelineRequest) (*CreatePipelineResponse, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/pipelines", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp CreatePipelineResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
