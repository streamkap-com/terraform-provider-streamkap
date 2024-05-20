package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type SourceResponse struct {
	Total    int      `json:"total"`
	PageSize int      `json:"page_size"`
	Page     int      `json:"page"`
	Result   []Source `json:"result"`
}

type Source struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connector string `json:"connector"`
	Config map[string]any `json:"config"`
}

type CreateSourceRequest struct {
	ID        string         `json:"-"`
	Name      *string        `json:"name"`
	Connector *string        `json:"connector"`
	Config    map[string]any `json:"config"`
}

type CreateSourceResponse struct {
	Cursor struct {
		TimestampFrom time.Time `json:"timestamp_from"`
		TimestampTo   time.Time `json:"timestamp_to"`
		TopicListFrom int       `json:"topic_list_from"`
		TopicListSize int       `json:"topic_list_size"`
		TopicListSort string    `json:"topic_list_sort"`
	} `json:"cursor"`
	EntityType string   `json:"entity_type"`
	Data       []Source `json:"data"`
}

func (s *streamkapAPI) CreateSource(ctx context.Context, reqPayload CreateSourceRequest) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/api/sources", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *streamkapAPI) GetSource(ctx context.Context, sourceID string) (*Source, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.BaseURL+"/api/sources?id="+sourceID, http.NoBody)
	if err != nil {
		return nil, err
	}
	var resp SourceResponse
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Result[0], nil
}

func (s *streamkapAPI) DeleteSource(ctx context.Context, sourceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.cfg.BaseURL+"/api/sources?id="+sourceID, http.NoBody)
	if err != nil {
		return err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return err
	}

	return nil
}

func (s *streamkapAPI) UpdateSource(ctx context.Context, reqPayload CreateSourceRequest) (*Source, error) {
	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, s.cfg.BaseURL+"/api/sources?id="+reqPayload.ID, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	var resp Source
	err = s.doRequest(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
